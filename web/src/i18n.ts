// Interface copy, in French and in English. Functional text is plain; the
// maritime world shows in names such as the logbook, never in instructions.
export type Lang = 'fr' | 'en';

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
  uptimeOver: (pct: string, days: number) => `${pct} sur ${days} jours`,
  noUptime: 'Pas encore mesuré',
  respondsIn: (ms: number) => `Répond en ${ms} ms`,
  daysAgo: (n: number) => `Il y a ${n} jours`,
  today: "Aujourd'hui",
  barLabel: (name: string, days: number, bad: number) =>
    `${name} : disponibilité jour par jour sur ${days} jours, ${bad === 0 ? 'sans interruption' : bad === 1 ? 'un jour avec interruption' : `${bad} jours avec interruption`}. Flèches gauche et droite pour lire un jour.`,
  dayFull: 'Aucune interruption',
  dayDown: (pct: string, min: string) => `${pct} de disponibilité, environ ${min} d'interruption`,
  dayNone: 'Pas de mesure ce jour-là',
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
  uptimeOver: (pct, days) => `${pct} over ${days} days`,
  noUptime: 'Not measured yet',
  respondsIn: (ms) => `Responds in ${ms} ms`,
  daysAgo: (n) => `${n} days ago`,
  today: 'Today',
  barLabel: (name, days, bad) =>
    `${name}: availability day by day over ${days} days, ${bad === 0 ? 'no interruption' : bad === 1 ? 'one day with an interruption' : `${bad} days with an interruption`}. Left and right arrows read a day.`,
  dayFull: 'No interruption',
  dayDown: (pct, min) => `${pct} available, about ${min} of interruption`,
  dayNone: 'No measurement that day',
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
