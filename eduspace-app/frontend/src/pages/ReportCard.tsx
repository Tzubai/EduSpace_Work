import { naira, useLoad } from "../api";
import { Msg } from "../ui";

export default function ReportCard({ studentId, termId, onClose }: { studentId: string; termId: string; onClose: () => void }) {
  const { data: r, error } = useLoad<any>(`/results/report-card/${studentId}?term_id=${termId}`);
  if (error) return <><Msg error={error} /><button className="small ghost" onClick={onClose}>Close</button></>;
  if (!r) return <p>Loading…</p>;
  return (
    <div className="paper">
      <div className="noprint row">
        <button className="primary" onClick={() => window.print()}>Print / Save as PDF</button>
        <button className="small ghost" onClick={onClose}>Close</button>
      </div>
      <h2 style={{ textAlign: "center" }}>{r.school}</h2>
      <h3 style={{ textAlign: "center" }}>Report card · {r.term}</h3>
      <p><b>{r.student}</b> · Admission no. {r.admission_number} · Class: {r.class || "—"}</p>
      <div className="scroll">
        <table>
          <thead><tr><th>Subject</th><th>Score</th><th>Out of</th><th>%</th><th>Grade</th><th>Remark</th></tr></thead>
          <tbody>
            {r.subjects.length === 0 && <tr><td colSpan={6} className="muted">No scores recorded yet.</td></tr>}
            {r.subjects.map((s: any) => (
              <tr key={s.subject}><td>{s.subject}</td><td>{s.score}</td><td>{s.max_score}</td><td>{s.percent}</td><td>{s.grade}</td><td>{s.remark}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
      <p><b>Average:</b> {r.average_percent}%</p>
      <p><b>Attendance:</b> {r.attendance.present} present, {r.attendance.late} late, {r.attendance.absent} absent</p>
      <p><b>Fees:</b> billed {naira(r.fees.billed_kobo)}, paid {naira(r.fees.paid_kobo)}, balance {naira(r.fees.balance_kobo)}</p>
    </div>
  );
}
