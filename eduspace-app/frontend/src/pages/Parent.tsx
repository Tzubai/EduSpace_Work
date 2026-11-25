import { useState } from "react";
import { api, naira, today, useLoad } from "../api";
import { Card, Msg, Select, Table } from "../ui";
import ReportCard from "./ReportCard";

export default function Parent({ terms, initialTab = "Overview" }: { terms: { id: string; name: string; session: string }[]; initialTab?: string }) {
  const kids = useLoad<any[]>("/parent/children");
  const [studentId, setStudentId] = useState("");
  const [termId, setTermId] = useState(terms[0]?.id || "");
  const [tab, setTab] = useState(initialTab);
  const [month, setMonth] = useState(today().slice(0, 7));
  const [report, setReport] = useState(false);

  const selected = Array.isArray(kids.data) ? kids.data.find((k) => k.student_id === studentId) || kids.data[0] : undefined;
  const sid = selected?.student_id || "";
  const suffix = sid ? `/parent/children/${sid}` : "";
  const summary = useLoad<any>(sid ? `${suffix}/summary?term_id=${termId}` : null);
  const invoices = useLoad<any[]>(sid ? `${suffix}/invoices?term_id=${termId}` : null);
  const payments = useLoad<any[]>(sid ? `${suffix}/payments?term_id=${termId}` : null);
  const attendance = useLoad<any[]>(sid ? `${suffix}/attendance?term_id=${termId}&month=${month}` : null);
  const timetable = useLoad<any[]>(sid ? `${suffix}/timetable?term_id=${termId}` : null);
  const results = useLoad<any[]>(sid ? `${suffix}/results?term_id=${termId}` : null);
  const announcements = useLoad<any[]>("/announcements");

  if (report && sid) return <ReportCard studentId={sid} termId={termId} onClose={() => setReport(false)} />;
  const tabs = ["Overview", "Results", "Attendance", "Fees", "Timetable", "Announcements", "School messages"];
  const x = summary.data || { attendance: { present: 0, absent: 0, late: 0, excused: 0 }, fees: { billed_kobo: 0, paid_kobo: 0, balance_kobo: 0 }, academic: { average_percent: 0 } };
  return (
    <div className="parent-portal">
      <div className="parent-head">
        <div>
          <p>PARENT PORTAL</p>
          <h1>{selected ? `${selected.first_name} ${selected.last_name}` : "My children"}</h1>
          <span>{selected?.admission_number ? `Admission no. ${selected.admission_number}` : "Linked child workspace"}</span>
        </div>
        <div className="parent-controls">
          {kids.data && kids.data.length > 1 && <Select placeholder="Choose child" value={sid} onChange={(e) => setStudentId(e.target.value)} options={kids.data.map((k: any) => ({ id: k.student_id, label: `${k.last_name} ${k.first_name}` }))} />}
          <Select placeholder="Term" value={termId} onChange={(e) => setTermId(e.target.value)} options={terms.map(t => ({ id: t.id, label: `${t.session} · ${t.name}` }))} />
        </div>
      </div>
      <Msg error={kids.error || summary.error || invoices.error || payments.error || attendance.error || timetable.error || results.error || announcements.error} />
      {!kids.data || kids.data.length === 0 ? <Card><p className="muted">No children are linked to your account yet. Please contact the school.</p></Card> : <>
        {initialTab === "Overview" && <div className="parent-tabs">{tabs.map(t => <button key={t} className={tab === t ? "active" : ""} onClick={() => setTab(t)}>{t}</button>)}</div>}
        {tab === "Overview" && <>
          <div className="parent-metrics">
            <article>
              <span>Academic average</span>
              <strong>{Math.round(Number(x.academic?.average_percent || 0))}%</strong>
              <small>Selected term</small>
            </article>
            <article>
              <span>Present</span>
              <strong>{x.attendance.present}</strong>
              <small>Recorded days</small>
            </article>
            <article>
              <span>Absent</span>
              <strong>{x.attendance.absent}</strong>
              <small>Recorded days</small>
            </article>
            <article>
              <span>Fee balance</span>
              <strong>{naira(Number(x.fees.balance_kobo || 0))}</strong>
              <small>Outstanding</small>
            </article>
          </div>
          <div className="parent-grid">
            <Card title="Child summary">
              <p><b>Attendance:</b> {x.attendance.present} present, {x.attendance.late} late, {x.attendance.absent} absent, {x.attendance.excused} excused</p>
              <p><b>Fees:</b> billed {naira(Number(x.fees.billed_kobo || 0))}, paid {naira(Number(x.fees.paid_kobo || 0))}</p>
              <button className="primary" onClick={() => setReport(true)}>Open report card</button>
            </Card>
            <Card title="Quick access">
              <div className="parent-actions">
                <button onClick={() => setTab("Results")}>▤ Results</button>
                <button onClick={() => setTab("Attendance")}>✓ Attendance</button>
                <button onClick={() => setTab("Fees")}>₦ Fees</button>
                <button onClick={() => setTab("Timetable")}>▣ Timetable</button>
              </div>
            </Card>
          </div>
        </>}
        {tab === "Results" && <Card title="Academic results">
          <Table head={["Subject", "Assessment", "Type", "Weight", "Score", "Out of", "%"]} rows={(results.data || []).map((r: any) => [r.subject, r.title, r.assessment_type, r.weight, r.score, r.max_score, `${r.percent}%`])} />
          <button className="primary" onClick={() => setReport(true)}>View printable report card</button>
        </Card>}
        {tab === "Attendance" && <>
          <Card title="Attendance history">
            <div className="row">
              <input type="month" value={month} onChange={e => setMonth(e.target.value)} />
            </div>
            <Table head={["Date", "Status", "Arrival", "Reason"]} rows={(attendance.data || []).map((r: any) => [r.day, r.status, r.arrival_time || "—", r.reason || "—"])} />
          </Card>
        </>}
        {tab === "Fees" && <div className="premium-grid">
          <Card title="Invoices">
            <Table head={["Term", "Description", "Amount", "Paid", "Balance", "Status"]} rows={(invoices.data || []).map((r: any) => [r.term, r.description, naira(r.amount_kobo), naira(r.paid_kobo), naira(r.balance_kobo), r.status])} />
          </Card>
          <Card title="Payment history">
            <Table head={["Date", "Description", "Method", "Reference", "Amount"]} rows={(payments.data || []).map((r: any) => [r.paid_at ? new Date(r.paid_at).toLocaleString() : "—", r.description, r.method, r.reference || "—", naira(r.amount_kobo)])} />
          </Card>
        </div>}
        {tab === "Timetable" && <Card title="Child timetable">
          <Table head={["Day", "Time", "Class", "Subject", "Teacher", "Room"]} rows={(timetable.data || []).map((r: any) => ["Mon Tue Wed Thu Fri Sat".split(" ")[Number(r.day_of_week) - 1], `${String(r.starts_at).slice(0, 5)}–${String(r.ends_at).slice(0, 5)}`, r.class_name, r.subject_name, r.teacher_name, r.room || "—"])} />
        </Card>}
        {tab === "Announcements" && <Card title="School announcements">
          <Table head={["Date", "Title", "Message"]} rows={(announcements.data || []).map((r: any) => [r.created_at ? new Date(r.created_at).toLocaleDateString() : "—", r.title, r.body])} />
        </Card>}
        {tab === "School messages" && <ParentMessages studentId={sid} />}
      </>}
    </div>
  );
}

function ParentMessages({ studentId }: { studentId: string }) {
  const [message, setMessage] = useState("");
  const [ok, setOk] = useState("");
  const [error, setError] = useState("");
  async function send() {
    setError("");
    setOk("");
    try {
      await api("POST", "/parent-messages/request", { student_id: studentId, message });
      setMessage("");
      setOk("Message sent to the school.")
    } catch (e: any) { setError(e.message) }
  }
  return (
    <Card title="Contact the school">
      <Msg error={error} ok={ok} />
      <textarea rows={5} placeholder="Write a message to the school" value={message} onChange={e => setMessage(e.target.value)} />
      <button className="primary" onClick={send}>Send message</button>
    </Card>
  );
}
