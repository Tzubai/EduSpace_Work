import { useEffect, useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, Select, useForm } from "../ui";
import { useMyClasses } from "./Attendance";

export default function Scores({ me, termId }: { me: { role: string }; termId: string }) {
  const { options } = useMyClasses(me.role);
  const subjects = useLoad<any[]>("/subjects");
  const [classId, setClassId] = useState("");
  const [asmId, setAsmId] = useState("");
  const asms = useLoad<any[]>(classId && termId ? `/results/assessments?class_id=${classId}&term_id=${termId}` : null);
  const grades = useLoad<any[]>(asmId ? `/results/grades?assessment_id=${asmId}` : null);
  const [scores, setScores] = useState<Record<string, string>>({});
  const f = useForm({ subject_id: "", title: "", max_score: "", assessment_type: "ca1", weight: "100" });
  const [error, setError] = useState("");
  const [ok, setOk] = useState("");

  useEffect(() => {
    const m: Record<string, string> = {};
    (grades.data || []).forEach((g) => (m[g.student_id] = g.score === null ? "" : String(g.score)));
    setScores(m);
  }, [grades.data]);

  async function createAsm() {
    setError(""); setOk("");
    try {
      await api("POST", "/results/assessments", { class_id: classId, term_id: termId, ...f.v, max_score: Number(f.v.max_score), weight: Number(f.v.weight) });
      f.reset(); asms.reload(); setOk("Assessment created.");
    } catch (e: any) { setError(e.message); }
  }
  async function save() {
    setError(""); setOk("");
    try {
      const list = Object.entries(scores).filter(([, v]) => v !== "").map(([student_id, v]) => ({ student_id, score: Number(v) }));
      await api("POST", "/results/grades", { assessment_id: asmId, scores: list });
      setOk("Scores saved.");
    } catch (e: any) { setError(e.message); }
  }

  const sel = (asms.data || []).find((a) => a.id === asmId);
  return (
    <>
      <Card title="Class">
        <Select placeholder="Choose class" value={classId} onChange={(e) => { setClassId(e.target.value); setAsmId(""); }} options={options} />
        <Msg error={error || asms.error || grades.error} ok={ok} />
      </Card>
      {classId && (
        <Card title="New test or exam">
          <div className="row">
            <Select placeholder="Subject" {...f.bind("subject_id")} options={(subjects.data || []).map((s) => ({ id: s.id, label: s.name }))} />
            <select {...f.bind("assessment_type")}>
              <option value="ca1">1st C.A.</option>
              <option value="ca2">2nd C.A.</option>
              <option value="exam">Exam</option>
              <option value="assignment">Assignment</option>
              <option value="project">Project</option>
              <option value="practical">Practical</option>
            </select>
            <input type="number" placeholder="Weight (%)" {...f.bind("weight")} />
            <input placeholder="Title (e.g. Test 1)" {...f.bind("title")} />
            <input type="number" placeholder="Maximum score" {...f.bind("max_score")} />
          </div>
          <button className="primary" onClick={createAsm}>Create</button>
        </Card>
      )}
      {classId && (
        <Card title="Enter scores">
          <Select placeholder="Choose test or exam" value={asmId} onChange={(e) => setAsmId(e.target.value)}
            options={(asms.data || []).map((a) => ({ id: a.id, label: `${a.subject} — ${a.title} (/${a.max_score})` }))} />
          {sel && (grades.data || []).map((g) => (
            <div className="row" key={g.student_id} style={{ justifyContent: "space-between", marginTop: 8 }}>
              <span>{g.last_name} {g.first_name}</span>
              <input type="number" style={{ flex: "0 0 100px" }} max={sel.max_score} min={0} value={scores[g.student_id] ?? ""}
                onChange={(e) => setScores({ ...scores, [g.student_id]: e.target.value })} />
            </div>
          ))}
          {sel && <button className="primary" onClick={save}>Save scores</button>}
        </Card>
      )}
    </>
  );
}
