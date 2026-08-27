'use client';

// Scheduler panel — the user-editable "business hours" feature: for every
// day of the week, the user enters their own opening/closing windows (any
// number of windows per day, or none = closed all day). VOID then computes,
// live, whether it is open right now and exactly how long remains until the
// next change — nothing about the schedule is assumed or hard-coded.

import React, { useEffect, useState } from 'react';
import { Clock, Plus, Save, Trash2 } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { api, ApiError } from '../../lib/api';
import type { BusinessHours, SchedulerStatus, Window } from '../../lib/types';

const DAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as const;

const DAY_LABELS: Record<string, Record<string, string>> = {
  en: { monday: 'Monday', tuesday: 'Tuesday', wednesday: 'Wednesday', thursday: 'Thursday', friday: 'Friday', saturday: 'Saturday', sunday: 'Sunday' },
  fa: { monday: 'دوشنبه', tuesday: 'سه‌شنبه', wednesday: 'چهارشنبه', thursday: 'پنجشنبه', friday: 'جمعه', saturday: 'شنبه', sunday: 'یکشنبه' },
  zh: { monday: '星期一', tuesday: '星期二', wednesday: '星期三', thursday: '星期四', friday: '星期五', saturday: '星期六', sunday: '星期日' },
};

function emptyHours(): BusinessHours {
  return { timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC', days: {} };
}

export default function SchedulerPanel() {
  const { t, locale } = useI18n();
  const [hours, setHours] = useState<BusinessHours>(emptyHours());
  const [status, setStatus] = useState<SchedulerStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [now, setNow] = useState(new Date());

  useEffect(() => {
    api.getHours().then((h) => setHours(h.days ? h : emptyHours())).catch(() => { /* start from a blank schedule */ });
  }, []);

  useEffect(() => {
    let cancelled = false;
    const poll = async () => {
      try {
        const s = await api.schedulerStatus();
        if (!cancelled) setStatus(s);
      } catch {
        /* keep last known status */
      }
    };
    poll();
    const id = setInterval(poll, 5000);
    const clock = setInterval(() => setNow(new Date()), 1000);
    return () => { cancelled = true; clearInterval(id); clearInterval(clock); };
  }, []);

  const addWindow = (day: string) =>
    setHours((h) => ({ ...h, days: { ...h.days, [day]: [...(h.days[day] ?? []), { start: '09:00', end: '17:00' }] } }));
  const updateWindow = (day: string, idx: number, patch: Partial<Window>) =>
    setHours((h) => ({
      ...h,
      days: { ...h.days, [day]: (h.days[day] ?? []).map((w, i) => (i === idx ? { ...w, ...patch } : w)) },
    }));
  const removeWindow = (day: string, idx: number) =>
    setHours((h) => ({ ...h, days: { ...h.days, [day]: (h.days[day] ?? []).filter((_, i) => i !== idx) } }));

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const saved = await api.setHours(hours);
      setHours(saved);
      const s = await api.schedulerStatus();
      setStatus(s);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to save business hours');
    } finally {
      setBusy(false);
    }
  };

  const dayLabels = DAY_LABELS[locale] ?? DAY_LABELS.en;

  return (
    <div className="void-scrollable" style={{ height: '100%', padding: 20 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
        <Clock size={18} color="var(--accent)" />
        <h2 style={{ margin: 0, fontSize: 17 }}>{t('scheduler.title')}</h2>
      </div>
      <p style={{ color: 'var(--fg-secondary)', fontSize: 13, maxWidth: 640, marginTop: 4 }}>{t('scheduler.description')}</p>

      {/* Live status card */}
      <div className="void-card" style={{ marginBottom: 18, display: 'flex', alignItems: 'center', gap: 18, flexWrap: 'wrap' }}>
        <span className={`void-badge ${status?.isOpen ? 'running' : 'stopped'}`}>
          <span className="void-badge-dot" />
          {status?.isOpen ? t('status.open') : t('status.closed')}
        </span>
        <div style={{ fontSize: 13 }}>
          <span style={{ color: 'var(--fg-tertiary)' }}>{t('label.timeUntilNext')}: </span>
          <span className="void-mono" style={{ fontWeight: 700 }}>{status?.timeUntilNextHuman ?? '—'}</span>
        </div>
        <div style={{ fontSize: 12, color: 'var(--fg-tertiary)' }} className="void-mono">
          {now.toLocaleString(locale === 'fa' ? 'fa-IR' : locale === 'zh' ? 'zh-CN' : 'en-US', { timeZone: hours.timezone || undefined })}
          {' · '}{hours.timezone}
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
        <label className="void-label">{t('label.timezone')}</label>
        <input className="void-input" style={{ width: 240 }} value={hours.timezone}
          onChange={(e) => setHours((h) => ({ ...h, timezone: e.target.value }))} placeholder="Europe/Berlin" />
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {DAYS.map((day) => {
          const windows = hours.days[day] ?? [];
          return (
            <div key={day} className="void-card" style={{ display: 'flex', gap: 14, alignItems: 'flex-start', flexWrap: 'wrap' }}>
              <div style={{ width: 110, fontWeight: 700, fontSize: 13, paddingTop: 6 }}>{dayLabels[day]}</div>
              <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6, minWidth: 240 }}>
                {windows.length === 0 && <div style={{ fontSize: 12, color: 'var(--fg-tertiary)', paddingTop: 6 }}>{t('scheduler.noWindows')}</div>}
                {windows.map((w, i) => (
                  <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <input className="void-input" type="time" value={w.start} onChange={(e) => updateWindow(day, i, { start: e.target.value })} />
                    <span style={{ color: 'var(--fg-tertiary)' }}>–</span>
                    <input className="void-input" type="time" value={w.end} onChange={(e) => updateWindow(day, i, { end: e.target.value })} />
                    <button className="void-icon-btn" onClick={() => removeWindow(day, i)}><Trash2 size={14} /></button>
                  </div>
                ))}
              </div>
              <button className="void-btn" onClick={() => addWindow(day)}><Plus size={14} /> {t('action.addWindow')}</button>
            </div>
          );
        })}
      </div>

      <div style={{ marginTop: 18 }}>
        <button className="void-btn void-btn-primary" onClick={save} disabled={busy}>
          <Save size={14} /> {t('action.save')}
        </button>
        {error && <span style={{ color: 'var(--danger)', fontSize: 12, marginInlineStart: 10 }}>{error}</span>}
      </div>
    </div>
  );
}
