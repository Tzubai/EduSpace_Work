import { ReactNode, useState } from "react";

export const Card = ({ title, children }: { title?: string; children: ReactNode }) => (
  <section className="card">
    {title && <h2>{title}</h2>}
    {children}
  </section>
);

export const Msg = ({ error, ok }: { error?: string; ok?: string }) =>
  error ? <p className="msg error">{error}</p> : ok ? <p className="msg ok">{ok}</p> : null;

export const Table = ({ head, rows }: { head: string[]; rows: ReactNode[][] }) => (
  <div className="scroll">
    <table>
      <thead>
        <tr>{head.map((h) => <th key={h}>{h}</th>)}</tr>
      </thead>
      <tbody>
        {rows.length === 0 && (
          <tr><td colSpan={head.length} className="muted">Nothing here yet.</td></tr>
        )}
        {rows.map((r, i) => (
          <tr key={i}>{r.map((c, j) => <td key={j}>{c}</td>)}</tr>
        ))}
      </tbody>
    </table>
  </div>
);

export function useForm<T extends Record<string, string>>(initial: T) {
  const [v, setV] = useState<T>(initial);
  const bind = (k: keyof T) => ({
    value: v[k],
    onChange: (e: any) => setV({ ...v, [k]: e.target.value }),
  });
  return { v, bind, reset: () => setV(initial), set: setV };
}

export const Select = ({ value, onChange, options, placeholder }: {
  value: string; onChange: (e: any) => void; options: { id: string; label: string }[]; placeholder: string;
}) => (
  <select value={value} onChange={onChange}>
    <option value="">{placeholder}</option>
    {options.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
  </select>
);
