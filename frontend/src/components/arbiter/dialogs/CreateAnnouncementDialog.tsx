'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { MessageSquare, Loader2 } from 'lucide-react';
import { useCreateAnnouncement } from '../hooks/useArbiterMutations';
import type { CreateAnnouncementDialogProps } from '../types';
import type { AnnouncementPriority } from '@/types/arbiter';

/**
 * Dialog for creating tournament announcements
 */
export function CreateAnnouncementDialog({ tournamentId }: CreateAnnouncementDialogProps) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [priority, setPriority] = useState<AnnouncementPriority>('info');
  const mutation = useCreateAnnouncement(tournamentId);

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setTitle('');
      setMessage('');
      setPriority('info');
    }
  }, [open]);

  const handleSubmit = () => {
    mutation.mutate(
      { title, message, priority },
      {
        onSuccess: () => setOpen(false),
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <MessageSquare className="mr-2 h-4 w-4" />
          Ankündigung
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Neue Ankündigung</DialogTitle>
          <DialogDescription>
            Erstellen Sie eine Ankündigung für alle Teilnehmer
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="announcement-title">Titel</Label>
            <Input
              id="announcement-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="z.B. Rundenbeginn"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="announcement-message">Nachricht</Label>
            <Textarea
              id="announcement-message"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Nachricht eingeben..."
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="announcement-priority">Priorität</Label>
            <Select
              value={priority}
              onValueChange={(v) => setPriority(v as AnnouncementPriority)}
            >
              <SelectTrigger id="announcement-priority">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="info">Information</SelectItem>
                <SelectItem value="warning">Warnung</SelectItem>
                <SelectItem value="urgent">Dringend</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!title || !message || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Veröffentlichen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
