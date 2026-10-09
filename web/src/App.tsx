import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { copy, initialLang, rememberLang, type Copy, type Lang } from './i18n';

type ServiceState = 'up' | 'down' | 'unknown';
type Overall = 'ok' | 'partial' | 'down' | 'unknown';

interface Day {
  date: string;
  uptime: number | null;
  downMin: number;
}
interface Service {
  id: string;
  name: string;
  description?: string;
  url?: string;
  state: ServiceState;
  responseMs: number;
  uptime: Record<string, number | null>;
  history: Day[];
}
interface Incident {
  id: number;
  service: string;
  start: string;
  end: string | null;
}
interface Notice {
  date: string;
  title: string;
  body?: string;
}
interface Status {
  title: string;
  demo?: boolean;
  interval: number;
  state: Overall;
  readAt: string | null;
  stale: boolean;
  days: number;
  services: Service[];
  incidents: Incident[];
  notices: Notice[];
}

// A day counts as interrupted from two minutes without an answer: a single
// failed check is not worth a mark. Half an hour separates brief from long.
const NOTICEABLE = 2;
const LONG = 30;

function dayKind(d: Day): 'none' | 'ok' | 'brief' | 'long' {
  if (d.uptime === null) return 'none';
  if (d.downMin < NOTICEABLE) return 'ok';
  return d.downMin < LONG ? 'brief' : 'long';
}

function useFormat(lang: Lang, t: Copy) {
  return useMemo(() => {
    const pct = new Intl.NumberFormat(lang, { style: 'percent', minimumFractionDigits: 2, maximumFractionDigits: 2 });
    const day = new Intl.DateTimeFormat(lang, { weekday: 'long', day: 'numeric', month: 'long' });
    const shortDay = new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'long', year: 'numeric' });
    const time = new Intl.DateTimeFormat(lang, { hour: '2-digit', minute: '2-digit' });
    const relative = new Intl.RelativeTimeFormat(lang, { numeric: 'always' });
    return {
      // Never rounds an imperfect figure up to 100 %.
      percent: (ratio: number) => pct.format(ratio < 1 ? Math.min(ratio, 0.9999) : 1),
      day: (date: string) => day.format(new Date(`${date}T12:00:00`)),
      date: (iso: string) => shortDay.format(new Date(iso)),
      time: (iso: string) => time.format(new Date(iso)),
      ago: (iso: string, now: number) => {
        const s = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
        if (s < 90) return relative.format(-s, 'second');
        if (s < 5400) return relative.format(-Math.round(s / 60), 'minute');
        if (s < 129600) return relative.format(-Math.round(s / 3600), 'hour');
        return relative.format(-Math.round(s / 86400), 'day');
      },
      duration: (minutes: number) => {
        if (minutes < 1) return t.lessThanMinute;
        if (minutes < 60) return `${Math.round(minutes)} ${t.min}`;
        const h = Math.floor(minutes / 60);
        const m = Math.round(minutes % 60);
        return m ? `${h} ${t.hour} ${String(m).padStart(2, '0')}` : `${h} ${t.hour}`;
      },
    };
  }, [lang, t]);
}
type Format = ReturnType<typeof useFormat>;

function useNarrow() {
  const query = '(max-width: 40rem)';
  const [narrow, setNarrow] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const on = () => setNarrow(media.matches);
    media.addEventListener('change', on);
    return () => media.removeEventListener('change', on);
  }, []);
  return narrow;
}

function Logo() {
  return (
    <svg className="logo" viewBox="0 0 64 64" aria-hidden>
      <rect width="64" height="64" rx="16" className="logo-tile" />
      <path d="M20 16v26a6 6 0 0 0 6 6h12" className="logo-trace" />
      <circle cx="46" cy="48" r="5.5" className="logo-node" />
    </svg>
  );
}

// State is never carried by colour alone: each one has its own glyph.
function Glyph({ state }: { state: ServiceState | Overall }) {
  const kind = state === 'ok' || state === 'up' ? 'up' : state === 'unknown' ? 'unknown' : state === 'partial' ? 'partial' : 'down';
  return (
    <svg className={`glyph glyph-${kind}`} viewBox="0 0 24 24" aria-hidden>
      <circle cx="12" cy="12" r="11" />
      {kind === 'up' && <path d="M7 12.5l3.2 3.2L17 8.8" />}
      {kind === 'down' && <path d="M8 8l8 8M16 8l-8 8" />}
      {kind === 'partial' && <path d="M12 6.5v7M12 17v.5" />}
      {kind === 'unknown' && <path d="M9.2 9.3a2.9 2.9 0 1 1 4.3 2.6c-.9.5-1.5 1.1-1.5 2.1M12 17.3v.2" />}
    </svg>
  );
}

// The sea under the headline. Its swell is the state of the lab: nearly flat
// when everything responds, rougher as services stop responding.
function Swell({ state }: { state: Overall }) {
  const amplitude = { ok: 5, unknown: 3, partial: 11, down: 17 }[state];
  const period = 160;
  const wave = (y: number, a: number) => {
    let d = `M${-period} ${y}`;
    for (let x = -period; x < 1600 + period; x += period) d += ` q${period / 4} ${-a} ${period / 2} 0 t${period / 2} 0`;
    return `${d} V64 H${-period} Z`;
  };
  return (
    <svg className={`swell swell-${state}`} viewBox="0 0 1600 64" preserveAspectRatio="xMinYMax slice" aria-hidden>
      <path className="swell-back" d={wave(26, amplitude * 0.8)} />
      <path className="swell-front" d={wave(36, amplitude)} />
    </svg>
  );
}

// The lab's cat, eye patch included, looking through a porthole.
function Mascot() {
  return (
    <svg className="mascot" viewBox="0 0 64 64" aria-hidden>
      <clipPath id="porthole">
        <circle cx="32" cy="32" r="27" />
      </clipPath>
      <circle cx="32" cy="32" r="27" className="mascot-sea" />
      <g clipPath="url(#porthole)">
        <g transform="translate(-7.6 10) scale(0.66)">
          <path d="M32 110C27 82 40 56 60 56s33 26 28 54z" className="mascot-fur" />
          <path d="M40 31 42 8l15 13z" className="mascot-fur" />
          <path d="M80 31 78 8 63 21z" className="mascot-fur mascot-ear" />
          <ellipse cx="60" cy="40" rx="25" ry="21" className="mascot-fur" />
          <ellipse cx="50" cy="40" rx="3.4" ry="4.6" className="mascot-ink mascot-eye" />
          <path d="M39.5 27.5 84.5 44" className="mascot-strap" />
          <path d="M63 34.5h14v6a7 7 0 0 1-14 0z" className="mascot-ink mascot-patch" />
          <path d="M57.5 47h5L60 50z" className="mascot-ink" />
          <path d="M43 57q17 9 34 0" className="mascot-collar" />
        </g>
      </g>
      <circle cx="32" cy="32" r="28.5" className="mascot-rim" />
    </svg>
  );
}

// A calm sea: the drawing for a logbook with nothing in it.
function CalmSea() {
  return (
    <svg className="art" viewBox="0 0 64 64" aria-hidden>
      <circle cx="32" cy="27" r="11" className="art-fill" />
      <circle cx="32" cy="27" r="11" />
      <path d="M32 7v3.5M14.5 14.5l2.5 2.5M49.5 14.5 47 17M9 30h3.5M51.5 30H55" />
      <path d="M7 45q6.25-5 12.5 0t12.5 0 12.5 0 12.5 0M13 54q6.33-5 12.67 0t12.66 0 12.67 0" className="art-accent" />
    </svg>
  );
}

function Bar({ service, days, t, f }: { service: Service; days: Day[]; t: Copy; f: Format }) {
  const [at, setAt] = useState<number | null>(null);
  const bad = days.filter((d) => ['brief', 'long'].includes(dayKind(d))).length;
  const day = at === null ? null : days[at];
  const move = (e: React.KeyboardEvent) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    setAt((i) => Math.min(days.length - 1, Math.max(0, (i ?? days.length - 1) + (e.key === 'ArrowLeft' ? -1 : 1))));
  };
  return (
    <div className="bar-wrap">
      <div
        className="bar"
        role="group"
        tabIndex={0}
        aria-label={t.barLabel(service.name, days.length, bad)}
        onKeyDown={move}
        onFocus={() => setAt((i) => i ?? days.length - 1)}
        onBlur={() => setAt(null)}
        onPointerLeave={() => setAt(null)}
      >
        {days.map((d, i) => (
          <i key={d.date} className={`cell cell-${dayKind(d)}${at === i ? ' cell-at' : ''}`} onPointerEnter={() => setAt(i)} onPointerDown={() => setAt(i)} />
        ))}
      </div>
      {day && at !== null && (
        <p className="tip" role="status" style={{ '--at': (at + 0.5) / days.length } as React.CSSProperties}>
          <strong>{f.day(day.date)}</strong>
          {day.uptime === null ? t.dayNone : day.downMin < NOTICEABLE ? t.dayFull : t.dayDown(f.percent(day.uptime), f.duration(day.downMin))}
        </p>
      )}
      <p className="bar-ends" aria-hidden>
        <span>{t.daysAgo(days.length)}</span>
        <span>{t.today}</span>
      </p>
    </div>
  );
}

function ServiceRow({ service, shown, t, f }: { service: Service; shown: number; t: Copy; f: Format }) {
  const days = service.history.slice(-shown);
  const uptime = service.uptime.all;
  return (
    <li className="service">
      <div className="service-head">
        <div className="service-name">
          <h3>
            {service.url ? (
              <a href={service.url} rel="noreferrer">
                {service.name}
              </a>
            ) : (
              service.name
            )}
          </h3>
          {service.description && <p>{service.description}</p>}
        </div>
        <div className="service-figures">
          <p className={`state state-${service.state}`}>
            <Glyph state={service.state} />
            {t.state[service.state]}
          </p>
          <p className="uptime">{uptime === null || uptime === undefined ? t.noUptime : t.uptimeOver(f.percent(uptime), service.history.length)}</p>
        </div>
      </div>
      <Bar service={service} days={days} t={t} f={f} />
    </li>
  );
}

function Logbook({ incidents, days, now, t, f }: { incidents: Incident[]; days: number; now: number; t: Copy; f: Format }) {
  const [all, setAll] = useState(false);
  const first = 6;
  const shown = all ? incidents : incidents.slice(0, first);
  if (incidents.length === 0) {
    return (
      <div className="calm">
        <CalmSea />
        <p>{t.logbookEmpty(days)}</p>
      </div>
    );
  }
  return (
    <>
      <ol className="log">
        {shown.map((in_) => {
          const end = in_.end ? new Date(in_.end).getTime() : now;
          const minutes = (end - new Date(in_.start).getTime()) / 60000;
          return (
            <li key={in_.id} className={in_.end ? 'entry' : 'entry entry-ongoing'}>
              <time dateTime={in_.start}>{f.date(in_.start)}</time>
              <div>
                <h3>{in_.service}</h3>
                <p>
                  {in_.end ? t.outage(f.duration(minutes)) : t.outageOngoing(f.duration(minutes))}, {in_.end ? t.fromTo(f.time(in_.start), f.time(in_.end)) : t.since(f.time(in_.start))}
                </p>
              </div>
              <p className={`state state-${in_.end ? 'up' : 'down'}`}>
                <Glyph state={in_.end ? 'up' : 'down'} />
                {in_.end ? t.resolved : t.ongoing}
              </p>
            </li>
          );
        })}
      </ol>
      {!all && incidents.length > first && (
        <button className="more" onClick={() => setAll(true)}>
          {t.older(incidents.length - first)}
        </button>
      )}
    </>
  );
}

export function App() {
  const [lang, setLang] = useState<Lang>(initialLang);
  const [status, setStatus] = useState<Status | null>(null);
  const [lost, setLost] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  const t = copy[lang];
  const f = useFormat(lang, t);
  const narrow = useNarrow();
  const timer = useRef(0);

  const load = useCallback(async () => {
    try {
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error(String(res.status));
      setStatus(await res.json());
      setLost(false);
    } catch {
      setLost(true);
    }
    setNow(Date.now());
  }, []);

  useEffect(() => {
    load();
    timer.current = window.setInterval(load, 30_000);
    const clock = window.setInterval(() => setNow(Date.now()), 10_000);
    return () => {
      clearInterval(timer.current);
      clearInterval(clock);
    };
  }, [load]);

  useEffect(() => {
    document.documentElement.lang = lang;
    document.title = `${t.pageTitle} · ${status?.title ?? 'lab.bingo'}`;
  }, [lang, t, status?.title]);

  const switchLang = () => {
    const next = lang === 'fr' ? 'en' : 'fr';
    setLang(next);
    rememberLang(next);
  };

  // Without an answer from the server the last known state cannot be trusted.
  const overall: Overall = !status || lost ? 'unknown' : status.state;
  const down = status?.services.filter((s) => s.state === 'down').length ?? 0;
  const headline = !status
    ? lost
      ? t.headline.unknown
      : t.loading
    : overall === 'ok'
      ? t.headline.ok
      : overall === 'down'
        ? t.headline.down
        : overall === 'partial'
          ? down === 1
            ? t.headline.one
            : t.headline.several(down)
          : t.headline.unknown;
  const reading = lost ? t.unreachable : !status ? '' : !status.readAt ? t.never : status.stale ? t.staleSince(f.ago(status.readAt, now)) : t.readAgo(f.ago(status.readAt, now));
  const shownDays = status ? (narrow ? Math.min(45, status.days) : status.days) : 0;

  return (
    <>
      <header className="sea">
        <div className="sea-top">
          <a className="lockup" href="/">
            <Logo />
            <span>
              {status?.title ?? 'lab.bingo'} <span className="lockup-kind">status</span>
            </span>
          </a>
          <button className="lang" onClick={switchLang} aria-label={t.switchLangLabel} lang={lang === 'fr' ? 'en' : 'fr'}>
            {t.switchLang}
          </button>
        </div>
        <div className="sea-body" aria-live="polite">
          <h1>
            <Glyph state={overall} />
            {headline}
          </h1>
          <p>{reading}</p>
        </div>
        <Swell state={overall} />
      </header>

      <main>
        {status?.demo && <p className="demo">{t.demo}</p>}

        {status && status.notices.length > 0 && (
          <section aria-labelledby="notices">
            <h2 id="notices">{t.notices}</h2>
            <ul className="notices">
              {status.notices.map((n, i) => (
                <li key={i}>
                  <time dateTime={n.date}>{f.date(`${n.date}T12:00:00`)}</time>
                  <div>
                    <h3>{n.title}</h3>
                    {n.body && <p>{n.body}</p>}
                  </div>
                </li>
              ))}
            </ul>
          </section>
        )}

        {status && (
          <section aria-labelledby="services">
            <h2 id="services">{t.services}</h2>
            <ul className="services">
              {status.services.map((s) => (
                <ServiceRow key={s.id} service={s} shown={shownDays} t={t} f={f} />
              ))}
            </ul>
            <ul className="legend" aria-hidden>
              {(['ok', 'brief', 'long', 'none'] as const).map((k) => (
                <li key={k}>
                  <i className={`cell cell-${k}`} />
                  {t.legend[k]}
                </li>
              ))}
            </ul>
          </section>
        )}

        {status && (
          <section aria-labelledby="logbook">
            <h2 id="logbook">{t.logbook}</h2>
            {status.incidents.length > 0 && <p className="lead">{t.logbookLead(status.days)}</p>}
            <Logbook incidents={status.incidents} days={status.days} now={now} t={t} f={f} />
          </section>
        )}
      </main>

      <footer className="foot">
        <Mascot />
        <p>{status ? t.foot(status.interval) : ''}</p>
      </footer>
    </>
  );
}
