// Interface copy, in French and in English. Functional text is plain; the
// maritime world shows in names such as the logbook, never in instructions.
export type Lang = 'fr' | 'en';
// The period the bars cover: a day and a week by the hour, more by the day.
export type Range = 'day' | 'week' | 'month' | 'all';

const fr = {
  pageTitle: 'État des services',
  switchLang: 'English',
  switchLangLabel: 'Read this page in English',
  headline: {
    ok: 'Tous les services répondent',
    one: 'Un service ne répond pas',
    several: (n: number) => `${n} services ne répondent pas`,
    down: 'Aucun service ne répond',
    unknown: 'Les mesures ne nous parviennent plus',
  },
  loading: 'Lecture des mesures',
  unreachable: 'Cette page ne parvient pas à lire les mesures. Elle réessaie toute seule.',
  readAgo: (ago: string) => `Dernier relevé ${ago}.`,
  staleSince: (ago: string) => `Dernier relevé ${ago} : l'état actuel des services n'est pas connu.`,
  never: "Aucun relevé n'a encore été fait.",
  demo: 'Données de démonstration : cet historique est inventé.',
  services: 'Services',
  state: { up: 'Répond', down: 'Ne répond pas', unknown: 'État inconnu' },
  rangeLabel: 'Période affichée',
  span: { day: () => '24 heures', week: () => '7 jours', month: () => '30 jours', all: (days: number) => `${days} jours` } as Record<Range, (days: number) => string>,
  spanShort: { day: () => '24 h', week: () => '7 j', month: () => '30 j', all: (days: number) => `${days} j` } as Record<Range, (days: number) => string>,
  uptimeOver: (pct: string, span: string) => `${pct} sur ${span}`,
  noUptime: 'Pas encore mesuré',
  respondsIn: (ms: number) => `Répond en ${ms} ms`,
  ago: (span: string) => `Il y a ${span}`,
  today: "Aujourd'hui",
  now: 'Maintenant',
  barLabel: (name: string, span: string, bad: number) =>
    `${name} : disponibilité sur ${span}, ${bad === 0 ? 'sans interruption' : bad === 1 ? 'une période avec interruption' : `${bad} périodes avec interruption`}. Flèches gauche et droite pour lire une période.`,
  dayFull: 'Aucune interruption',
  dayDown: (pct: string, min: string) => `${pct} de disponibilité, environ ${min} d'interruption`,
  dayNone: 'Pas de mesure ce jour-là',
  spanNone: 'Pas de mesure sur cette période',
  legend: { ok: 'Sans interruption', brief: "Moins de 30 min d'interruption", long: '30 min ou plus', none: 'Pas de mesure' },
  logbook: 'Journal de bord',
  logbookLead: (days: number) => `Les interruptions relevées sur les ${days} derniers jours.`,
  logbookEmpty: (days: number) => `Aucune interruption relevée sur les ${days} derniers jours.`,
  ongoing: 'En cours',
  resolved: 'Résolu',
  outage: (duration: string) => `Interruption de ${duration}`,
  outageOngoing: (duration: string) => `Ne répond plus depuis ${duration}`,
  fromTo: (from: string, to: string) => `de ${from} à ${to}`,
  since: (from: string) => `début à ${from}`,
  older: (n: number) => (n === 1 ? "Afficher l'interruption plus ancienne" : `Afficher les ${n} interruptions plus anciennes`),
  notices: 'Annonces',
  foot: (seconds: number) => `Chaque service est vérifié toutes les ${seconds} secondes depuis le lab.`,
  min: 'min',
  hour: 'h',
  lessThanMinute: "moins d'une minute",
};

const en: typeof fr = {
  pageTitle: 'Service status',
  switchLang: 'Français',
  switchLangLabel: 'Lire cette page en français',
  headline: {
    ok: 'All services are responding',
    one: 'One service is not responding',
    several: (n) => `${n} services are not responding`,
    down: 'No service is responding',
    unknown: 'Measurements are not reaching us',
  },
  loading: 'Reading measurements',
  unreachable: 'This page cannot read the measurements. It retries by itself.',
  readAgo: (ago) => `Last reading ${ago}.`,
  staleSince: (ago) => `Last reading ${ago}: the current state of the services is not known.`,
  never: 'No reading has been made yet.',
  demo: 'Demonstration data: this history is made up.',
  services: 'Services',
  state: { up: 'Responding', down: 'Not responding', unknown: 'State unknown' },
  rangeLabel: 'Period shown',
  span: { day: () => '24 hours', week: () => '7 days', month: () => '30 days', all: (days) => `${days} days` },
  spanShort: { day: () => '24 h', week: () => '7 d', month: () => '30 d', all: (days) => `${days} d` },
  uptimeOver: (pct, span) => `${pct} over ${span}`,
  noUptime: 'Not measured yet',
  respondsIn: (ms) => `Responds in ${ms} ms`,
  ago: (span) => `${span} ago`,
  today: 'Today',
  now: 'Now',
  barLabel: (name, span, bad) =>
    `${name}: availability over ${span}, ${bad === 0 ? 'no interruption' : bad === 1 ? 'one period with an interruption' : `${bad} periods with an interruption`}. Left and right arrows read a period.`,
  dayFull: 'No interruption',
  dayDown: (pct, min) => `${pct} available, about ${min} of interruption`,
  dayNone: 'No measurement that day',
  spanNone: 'No measurement over that period',
  legend: { ok: 'No interruption', brief: 'Less than 30 min of interruption', long: '30 min or more', none: 'No measurement' },
  logbook: 'Logbook',
  logbookLead: (days) => `Interruptions recorded over the last ${days} days.`,
  logbookEmpty: (days) => `No interruption recorded over the last ${days} days.`,
  ongoing: 'Ongoing',
  resolved: 'Resolved',
  outage: (duration) => `Interruption of ${duration}`,
  outageOngoing: (duration) => `Not responding for ${duration}`,
  fromTo: (from, to) => `from ${from} to ${to}`,
  since: (from) => `started at ${from}`,
  older: (n) => (n === 1 ? 'Show the older interruption' : `Show the ${n} older interruptions`),
  notices: 'Announcements',
  foot: (seconds) => `Each service is checked every ${seconds} seconds from the lab.`,
  min: 'min',
  hour: 'h',
  lessThanMinute: 'less than a minute',
};

export const copy = { fr, en };
export type Copy = typeof fr;

export function initialLang(): Lang {
  try {
    const saved = localStorage.getItem('status-lang');
    if (saved === 'fr' || saved === 'en') return saved;
  } catch {
    // Storage may be blocked; the browser language decides.
  }
  return navigator.language.toLowerCase().startsWith('fr') ? 'fr' : 'en';
}

export function rememberLang(lang: Lang) {
  try {
    localStorage.setItem('status-lang', lang);
  } catch {
    // Nothing to remember without storage.
  }
}
