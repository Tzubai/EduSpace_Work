import { useEffect, useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, Select, Table, useForm } from "../ui";

function NameList({ title, path, onChange }: { title: string; path: string; onChange?: () => void }) {
  const list = useLoad<any[]>(path);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  async function add() {
    setError("");
    try { await api("POST", path, { name }); setName(""); list.reload(); onChange?.(); } catch (e: any) { setError(e.message); }
  }
  return (
    <Card title={title}>
      <div className="row">
        <input placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
        <button className="primary" onClick={add}>Add</button>
      </div>
      <Msg error={error} />
      <p className="muted">{(list.data || []).map((x) => x.name).join(" · ") || "None yet."}</p>
    </Card>
  );
}

export default function Setup({ termId, reloadTerms }: { termId: string; reloadTerms: () => void }) {
  const classes = useLoad<any[]>("/classes");
  const subjects = useLoad<any[]>("/subjects");
  const sessions = useLoad<any[]>("/sessions");
  const terms = useLoad<any[]>("/terms");
  const students = useLoad<any[]>("/students");
  const users = useLoad<any[]>("/users");
  const assignments = useLoad<any[]>("/teacher-assignments");
  const scale = useLoad<any[]>("/grading-scale");
  const [bands, setBands] = useState<any[]>([]);
  useEffect(() => { if (scale.data) setBands(scale.data); }, [scale.data]);

  const t = useForm({ session_id: "", name: "", starts_on: "", ends_on: "" });
  const e = useForm({ student_id: "", class_id: "" });
  const a = useForm({ user_id: "", class_id: "", subject_id: "" });
  const [error, setError] = useState("");
  const [ok, setOk] = useState("");

  async function run(fn: () => Promise<any>, after: () => void) {
    setError(""); setOk("");
    try { await fn(); setOk("Saved."); after(); } catch (x: any) { setError(x.message); }
  }
  const opts = (rows: any[] | null, label: (r: any) => string) => (rows || []).map((r) => ({ id: r.id, label: label(r) }));

  return (
    <>
      <Msg error={error} ok={ok} />
      <NameList title="Classes" path="/classes" onChange={classes.reload} />
      <NameList title="Subjects" path="/subjects" onChange={subjects.reload} />
      <NameList title="Academic sessions (e.g. 2026/2027)" path="/sessions" onChange={sessions.reload} />
      <Card title="Terms">
        <div className="row">
          <Select placeholder="Session" {...t.bind("session_id")} options={opts(sessions.data, (s) => s.name)} />
          <input placeholder="Term name (e.g. First Term)" {...t.bind("name")} />
          <input type="date" {...t.bind("starts_on")} />
          <input type="date" {...t.bind("ends_on")} />
        </div>
        <button className="primary" onClick={() => run(() => api("POST", "/terms", t.v), () => { t.reset(); terms.reload(); reloadTerms(); })}>Add term</button>
        <p className="muted">{(terms.data || []).map((x) => `${x.session} ${x.name}`).join(" · ")}</p>
      </Card>
      <Card title="Place a student in a class (for the selected term)">
        <div className="row">
          <Select placeholder="Student" {...e.bind("student_id")} options={opts(students.data, (s) => `${s.last_name} ${s.first_name}`)} />
          <Select placeholder="Class" {...e.bind("class_id")} options={opts(classes.data, (c) => c.name)} />
        </div>
        <button className="primary" disabled={!termId}
          onClick={() => run(() => api("POST", "/enrollments", { ...e.v, term_id: termId }), () => e.reset())}>Enrol</button>
      </Card>
      <Card title="Assign a teacher to a class">
        <div className="row">
          <Select placeholder="Teacher" {...a.bind("user_id")} options={opts((users.data || []).filter((u) => u.role === "teacher"), (u) => u.full_name)} />
          <Select placeholder="Class" {...a.bind("class_id")} options={opts(classes.data, (c) => c.name)} />
          <Select placeholder="All subjects" {...a.bind("subject_id")} options={opts(subjects.data, (s) => s.name)} />
        </div>
        <button className="primary" onClick={() => run(() => api("POST", "/teacher-assignments", a.v), () => { a.reset(); assignments.reload(); })}>Assign</button>
        <Table head={["Teacher", "Class", "Subject"]} rows={(assignments.data || []).map((x) => [x.teacher, x.class, x.subject])} />
      </Card>
      <Card title="Grading scale">
        <p className="muted">A score at or above the minimum gets that grade.</p>
        {bands.map((b, i) => (
          <div className="row" key={i}>
            <input type="number" value={b.min_score} onChange={(ev) => setBands(bands.map((x, j) => j === i ? { ...x, min_score: Number(ev.target.value) } : x))} />
            <input value={b.grade} onChange={(ev) => setBands(bands.map((x, j) => j === i ? { ...x, grade: ev.target.value } : x))} />
            <input value={b.remark} onChange={(ev) => setBands(bands.map((x, j) => j === i ? { ...x, remark: ev.target.value } : x))} />
          </div>
        ))}
        <button className="primary" onClick={() => run(() => api("PUT", "/grading-scale", { bands }), () => scale.reload())}>Save scale</button>
      </Card>
    </>
  );
}
