// aqaa load test: the Next.js frontend of OpenPairing.
// All pages are client components; the dynamic routes render their shell for any id,
// the data itself is fetched by the browser from the backend API.
import http from 'k6/http';
import { check } from 'k6';

const BASE = (__ENV.AQAA_APP_FRONTEND_URL || 'http://localhost:3000').replace(/\/+$/, '');
const VUS = parseInt(__ENV.AQAA_LOAD_VUS || '10', 10);
const DURATION = __ENV.AQAA_LOAD_DURATION || '30s';

export const options = {
  scenarios: {
    load: {
      executor: 'constant-vus',
      vus: Math.min(Math.max(VUS, 1), 10),
      duration: DURATION,
    },
  },
};

const TOURNAMENT_ID = '00000000-0000-4000-8000-000000000001';
const PLAYER_ID = '00000000-0000-4000-8000-000000000002';
const GAME_ID = '00000000-0000-4000-8000-000000000003';
const SLUG = 'aqaa-open';

const ENDPOINTS = [
  { name: 'frontend GET /', path: '/', weight: 10 },
  { name: 'frontend GET /tournaments', path: '/tournaments', weight: 10 },
  { name: 'frontend GET /tournaments/<id>', path: `/tournaments/${TOURNAMENT_ID}`, weight: 10 },
  { name: 'frontend GET /t/<slug>', path: `/t/${SLUG}`, weight: 10 },
  { name: 'frontend GET /arbiter/<tournamentId>', path: `/arbiter/${TOURNAMENT_ID}`, weight: 6 },
  { name: 'frontend GET /players', path: '/players', weight: 6 },
  { name: 'frontend GET /players/<id>', path: `/players/${PLAYER_ID}`, weight: 5 },
  { name: 'frontend GET /login', path: '/login', weight: 6 },
  { name: 'frontend GET /register', path: '/register', weight: 3 },
  { name: 'frontend GET /forgot-password', path: '/forgot-password', weight: 2 },
  { name: 'frontend GET /home', path: '/home', weight: 4 },
  { name: 'frontend GET /tournaments/new', path: '/tournaments/new', weight: 3 },
  { name: 'frontend GET /tournaments/<id>/register', path: `/tournaments/${TOURNAMENT_ID}/register`, weight: 3 },
  { name: 'frontend GET /tournaments/<id>/settings', path: `/tournaments/${TOURNAMENT_ID}/settings`, weight: 3 },
  { name: 'frontend GET /t/<slug>/check-in', path: `/t/${SLUG}/check-in`, weight: 3 },
  { name: 'frontend GET /play/<gameId>', path: `/play/${GAME_ID}`, weight: 3 },
  { name: 'frontend GET /profile', path: '/profile', weight: 2 },
  { name: 'frontend GET /settings', path: '/settings', weight: 2 },
];

const TOTAL = ENDPOINTS.reduce((s, e) => s + e.weight, 0);

function pick() {
  let r = Math.random() * TOTAL;
  for (const e of ENDPOINTS) {
    r -= e.weight;
    if (r < 0) return e;
  }
  return ENDPOINTS[ENDPOINTS.length - 1];
}

export default function () {
  const e = pick();
  const res = http.get(`${BASE}${e.path}`, { tags: { endpoint: e.name } });
  check(res, { 'status is 200': (r) => r.status === 200 }, { endpoint: e.name });
}
