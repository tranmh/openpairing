# OpenPairing - Overall Implementation Plan

## Executive Summary

**OpenPairing** is a modern, cloud-native chess tournament management SaaS platform targeting the DACH region (Germany, Austria, Switzerland). It competes with Tornelo while offering superior federation integration with DSB, ÖSB, and FIDE.

| Metric | Target |
|--------|--------|
| MVP | Month 6 |
| Beta | Month 7 |
| Launch | Month 8 |
| Test Coverage | 80-95% |

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go (Gin, Gorilla WebSocket) |
| Frontend | React, Next.js 14, TypeScript |
| Database | PostgreSQL |
| Cache/Realtime | Redis |
| Pairing Engines | JaVaFo (Java), bbpPairings (C++) |
| Payments | Stripe + SEPA |
| License | AGPL-3.0 |

---

## Architecture Overview

```
                         ┌─────────────────┐
                         │   CloudFlare    │
                         │      CDN        │
                         └────────┬────────┘
                                  │
                         ┌────────┴────────┐
                         │   Next.js App   │
                         │   (Frontend)    │
                         └────────┬────────┘
                                  │
                         ┌────────┴────────┐
                         │  API Gateway    │
                         └────────┬────────┘
                                  │
        ┌─────────────────────────┼─────────────────────────┐
        │                         │                         │
        ▼                         ▼                         ▼
┌───────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Core API    │       │  Chess Server   │       │  Worker Service │
│     (Go)      │       │  (WebSocket)    │       │   (Background)  │
└───────┬───────┘       └────────┬────────┘       └────────┬────────┘
        │                        │                         │
        └────────────────────────┼─────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
        ┌──────────┐      ┌──────────┐      ┌──────────┐
        │PostgreSQL│      │  Redis   │      │ S3/MinIO │
        └──────────┘      └──────────┘      └──────────┘
```

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-4)
**Goal:** Project setup, authentication, CI/CD

| Week | Focus | Deliverables |
|------|-------|--------------|
| 1 | Project Setup | Go module, Next.js, Docker, DB migrations |
| 2 | Auth Backend | JWT, RBAC, password reset, rate limiting |
| 3 | Auth Frontend | Login/register pages, auth context |
| 4 | Infrastructure | CI/CD, Dockerfiles, staging environment |

**Exit Criteria:**
- Users can register, login, manage profile
- 80% test coverage on auth module
- CI pipeline passing

---

### Phase 2: Player Database (Weeks 5-7)
**Goal:** Player management with federation sync

| Week | Focus | Deliverables |
|------|-------|--------------|
| 5 | Player Core | CRUD, search, profiles |
| 6 | FIDE Integration | XML import, monthly sync |
| 7 | DSB Integration | API client, DWZ sync |

**Exit Criteria:**
- FIDE import works (~500k players)
- DSB API integration complete
- Player search < 100ms

---

### Phase 3: Tournament Core (Weeks 8-12)
**Goal:** Full OTB tournament workflow

| Week | Focus | Deliverables |
|------|-------|--------------|
| 8 | Tournament CRUD | Creation wizard, settings |
| 9 | Registration | Participant management, CSV import |
| 10 | Pairing Engine | JaVaFo integration, TRF format |
| 11 | Results | Entry, tiebreaks (15+), standings |
| 12 | Polish | Complete workflow, public views |

**Exit Criteria:**
- JaVaFo pairings match FIDE test cases
- Run complete 7-round Swiss tournament
- TRF export validates

---

### Phase 4: Arbiter Tools (Weeks 13-14)
**Goal:** Professional arbiter experience

| Week | Focus | Deliverables |
|------|-------|--------------|
| 13 | Arbiter Console | Dashboard, status, quick entry |
| 14 | Arbiter Actions | Override, withdrawal, bye, audit |

**Exit Criteria:**
- All actions have audit logs
- Multi-arbiter concurrency safe
- Mobile view functional

---

### Phase 5: Chess Server (Weeks 15-18)
**Goal:** Online game play

| Week | Focus | Deliverables |
|------|-------|--------------|
| 15 | WebSocket | Connection management, auth, reconnect |
| 16 | Game Logic | Move validation, state, PGN |
| 17 | Clock | Timing, sync, timeout |
| 18 | Features | Draw, resign, abort, spectating |

**Exit Criteria:**
- All chess rules correct
- 100 concurrent games
- Move latency < 100ms

---

### Phase 6: Hybrid & Team Tournaments (Weeks 19-22)
**Goal:** Online+OTB hybrid, Mannschaftsmeisterschaft

| Week | Focus | Deliverables |
|------|-------|--------------|
| 19 | Hybrid | Link games to pairings, auto-sync |
| 20 | Team Core | Team CRUD, roster, registration |
| 21 | Team Match | Pairing, lineups, results |
| 22 | Team Polish | Standings, tiebreaks, Bundesliga |

**Exit Criteria:**
- Lineup validation follows German rules
- Complete 8-team tournament
- Team tiebreaks working

---

### Phase 7: Reporting & Submission (Weeks 23-24)
**Goal:** Federation integration

| Week | Focus | Deliverables |
|------|-------|--------------|
| 23 | Exports | TRF16, DWZ, PDF |
| 24 | Submission | DSB API, Chess-Results |

**Exit Criteria:**
- TRF passes FIDE validation
- DWZ accepted by DSB

---

### Phase 8: Polish & Beta (Weeks 25-28)
**Goal:** Production ready

| Week | Focus | Deliverables |
|------|-------|--------------|
| 25 | Features | Remaining tiebreaks, bbpPairings |
| 26 | Payment | Stripe, SEPA, notifications |
| 27 | Security | Audit, pentest, GDPR |
| 28 | Beta | Documentation, onboarding |

**Exit Criteria:**
- Security audit passed
- 5 real tournaments completed
- 99.9% uptime

---

## Key Modules

| Module | Description | Priority |
|--------|-------------|----------|
| Auth | JWT, RBAC, multi-language | P0 |
| Players | FIDE/DSB/ÖSB sync, unified profiles | P0 |
| Tournaments | Swiss, Round Robin, Knockout | P0 |
| Pairing | JaVaFo, bbpPairings, TRF | P0 |
| Results | Entry, tiebreaks (90+), standings | P0 |
| Chess Server | WebSocket, real-time games | P1 |
| Arbiter | Console, overrides, audit | P1 |
| Team | Mannschaftsmeisterschaft | P1 |
| Export | TRF, DWZ, PDF | P1 |
| Fair Play | Basic detection | P2 |

---

## Database Tables

| Category | Tables |
|----------|--------|
| Auth | users, organization_members |
| Players | players, clubs, rating_history |
| Tournaments | tournaments, tournament_sections, tournament_participants, tournament_rounds, games, forbidden_pairs |
| Teams | teams, team_members, team_tournament_participants, team_lineups, team_matches, team_match_boards, team_tournament_settings |
| Chess Server | chess_games |
| System | organizations, audit_log, notifications, federation_sync_log |

---

## API Endpoints Summary

| Category | Count | Examples |
|----------|-------|----------|
| Auth | 6 | /auth/login, /auth/register |
| Players | 8 | /players, /players/import/fide |
| Tournaments | 10 | /tournaments, /tournaments/:id/standings |
| Pairings | 6 | /rounds/:num/pair |
| Results | 4 | /games/:gid/result |
| Teams | 17 | /teams, /lineups, /matches |
| Chess | 5 | /games/:id/resign |
| Export | 5 | /reports/trf, /submit/dsb |

**Total: ~70+ endpoints**

---

## Test Strategy

### Testing Pyramid
```
         E2E (~50)
        ─────────
      Integration (~150)
     ─────────────────
        Unit (~500+)
```

### Coverage Targets
| Component | Target |
|-----------|--------|
| Auth | 90% |
| Tournament | 90% |
| Chess Server | 95% |
| Players | 85% |
| Teams | 85% |
| Export | 80% |

### Quality Gates
- Unit tests: 80% coverage minimum
- Integration tests: Required
- E2E tests: Required
- Security scan: No high/critical issues
- Lint: Must pass

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| JaVaFo issues | High | bbpPairings backup |
| DSB API changes | Medium | Abstract federation layer |
| 500+ player perf | Medium | Early load testing |
| FIDE endorsement | High | Focus OTB first |
| Chess scaling | High | Redis cluster |

---

## Success Criteria

### MVP (Month 6)
- [ ] 50-player Swiss OTB
- [ ] 8-team Mannschaftsturnier
- [ ] FIDE/DSB import
- [ ] TRF export
- [ ] 15+ tiebreaks
- [ ] 80% coverage

### Beta (Month 7)
- [ ] Hybrid tournaments
- [ ] 100 concurrent games
- [ ] 90+ tiebreaks
- [ ] German localization
- [ ] 5 beta customers

### Launch (Month 8)
- [ ] FIDE endorsement submitted
- [ ] Payments live
- [ ] 99.9% uptime
- [ ] 20+ organizations
- [ ] GDPR compliant

---

## File Structure

```
openpairing/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go
│   │   ├── worker/main.go
│   │   └── chess/main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── player/
│   │   ├── tournament/
│   │   ├── pairing/
│   │   ├── chess/
│   │   ├── team/
│   │   └── ...
│   ├── pkg/
│   ├── migrations/
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   ├── components/
│   │   ├── lib/
│   │   └── i18n/
│   └── Dockerfile
├── docker-compose.yml
├── docs/
└── README.md
```

---

## References

- **Detailed Plan:** `~/.claude/plans/agile-baking-creek.md`
- **FIDE Handbook:** https://handbook.fide.com/
- **DSB API:** https://www.schachbund.de/api.html
- **JaVaFo:** https://www.rrweb.org/javafo/JaVaFo.htm
- **bbpPairings:** https://github.com/BieremaBoyzProgramming/bbpPairings
