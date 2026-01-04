package fide

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// FIDE rating list URLs
	StandardRatingsURL = "https://ratings.fide.com/download/standard_rating_list_xml.zip"
	RapidRatingsURL    = "https://ratings.fide.com/download/rapid_rating_list_xml.zip"
	BlitzRatingsURL    = "https://ratings.fide.com/download/blitz_rating_list_xml.zip"

	// Alternative URLs (sometimes more reliable)
	PlayersListXMLURL = "https://ratings.fide.com/download/players_list_xml.zip"

	// Monthly list URL pattern
	// Example: https://ratings.fide.com/download/standard_jan25frl_xml.zip
	MonthlyListPattern = "https://ratings.fide.com/download/standard_%s%dfrl_xml.zip"
)

// Client handles downloads from FIDE
type Client struct {
	httpClient *http.Client
	logger     zerolog.Logger
	cacheDir   string
}

// NewClient creates a new FIDE download client
func NewClient(logger zerolog.Logger, cacheDir string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // FIDE files can be large
		},
		logger:   logger.With().Str("component", "fide_client").Logger(),
		cacheDir: cacheDir,
	}
}

// DownloadResult contains information about the downloaded file
type DownloadResult struct {
	FilePath    string
	Size        int64
	LastModified time.Time
	Checksum    string
}

// DownloadLatestRatings downloads the latest FIDE rating list
func (c *Client) DownloadLatestRatings(ratingType string) (*DownloadResult, error) {
	var url string
	switch ratingType {
	case "standard":
		url = StandardRatingsURL
	case "rapid":
		url = RapidRatingsURL
	case "blitz":
		url = BlitzRatingsURL
	default:
		url = PlayersListXMLURL
	}

	return c.downloadFile(url, fmt.Sprintf("fide_%s_ratings.zip", ratingType))
}

// DownloadMonthlyRatings downloads ratings for a specific month
func (c *Client) DownloadMonthlyRatings(year int, month time.Month) (*DownloadResult, error) {
	monthStr := month.String()[:3]
	monthStr = string(monthStr[0]) + string(monthStr[1]) + string(monthStr[2])
	yearShort := year % 100

	url := fmt.Sprintf(MonthlyListPattern, monthStr, yearShort)
	filename := fmt.Sprintf("fide_standard_%s%02d.zip", monthStr, yearShort)

	return c.downloadFile(url, filename)
}

func (c *Client) downloadFile(url, filename string) (*DownloadResult, error) {
	c.logger.Info().Str("url", url).Msg("Downloading FIDE rating list")

	// Ensure cache directory exists
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	filePath := filepath.Join(c.cacheDir, filename)

	// Check if file exists and is recent (less than 24 hours old)
	if info, err := os.Stat(filePath); err == nil {
		if time.Since(info.ModTime()) < 24*time.Hour {
			c.logger.Info().Str("file", filePath).Msg("Using cached FIDE file")
			return &DownloadResult{
				FilePath:     filePath,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			}, nil
		}
	}

	// Download file
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "OpenPairing/1.0 (+https://github.com/openpairing/openpairing)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy response body to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	c.logger.Info().
		Int64("bytes", written).
		Str("file", filePath).
		Msg("Downloaded FIDE file")

	// Parse last modified from response
	lastModified := time.Now()
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := time.Parse(time.RFC1123, lm); err == nil {
			lastModified = t
		}
	}

	return &DownloadResult{
		FilePath:     filePath,
		Size:         written,
		LastModified: lastModified,
	}, nil
}

// ExtractZip extracts a zip file and returns the path to the extracted XML file
func (c *Client) ExtractZip(zipPath string) (string, error) {
	c.logger.Info().Str("zip", zipPath).Msg("Extracting FIDE zip file")

	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	dir := filepath.Dir(zipPath)
	var xmlPath string

	// Find and extract the XML file
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Look for XML files
		if strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			c.logger.Info().
				Str("file", f.Name).
				Int64("size", int64(f.UncompressedSize64)).
				Msg("Found XML file in zip")

			// Extract the XML file
			xmlPath = filepath.Join(dir, filepath.Base(f.Name))

			// Open the file inside the zip
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open file in zip: %w", err)
			}

			// Create the output file
			outFile, err := os.Create(xmlPath)
			if err != nil {
				rc.Close()
				return "", fmt.Errorf("failed to create output file: %w", err)
			}

			// Copy the contents
			written, err := io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()

			if err != nil {
				return "", fmt.Errorf("failed to extract file: %w", err)
			}

			c.logger.Info().
				Str("file", xmlPath).
				Int64("bytes", written).
				Msg("Extracted XML file")

			// Return the first XML file found
			return xmlPath, nil
		}
	}

	return "", fmt.Errorf("no XML file found in zip archive")
}

// StreamReader creates a streaming reader for large XML files
type StreamReader struct {
	file   *os.File
	reader io.Reader
}

// NewStreamReader opens a file for streaming, handling gzip if needed
func NewStreamReader(path string) (*StreamReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Check if gzipped
	var reader io.Reader = file
	if filepath.Ext(path) == ".gz" {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		reader = gzReader
	}

	return &StreamReader{
		file:   file,
		reader: reader,
	}, nil
}

// Read implements io.Reader
func (r *StreamReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Close closes the stream reader
func (r *StreamReader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		closer.Close()
	}
	return r.file.Close()
}
