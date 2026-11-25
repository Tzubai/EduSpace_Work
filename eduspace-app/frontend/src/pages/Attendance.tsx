import { useEffect, useState } from "react";
import { api, today, useLoad } from "../api";
import { Card, Msg, Select, Table } from "../ui";

// Classes a person may work with: teachers see their assigned classes, staff see all.
export function useMyClasses(role: string) {
  const path = role === "teacher" ? "/teacher-assignments" : "/classes";
  const { data, error } = useLoad<any[]>(path);
  const seen = new Set<string>();
  const options = (data || [])
    .map((x) => ({ id: x.class_id || x.id, label: x.class || x.name }))
    .filter((o) => (seen.has(o.id) ? false : (seen.add(o.id), true)));
  return { options, error };
}

export default function Attendance({ me, termId }: { me: { role: string }; termId: string }) {
  const { options, error: clsError } = useMyClasses(me.role);
  const [classId, setClassId] = useState("");
  const [day, setDay] = useState(today());
  const [month, setMonth] = useState(today().slice(0,7));
  const roster = useLoad<any[]>(classId && termId ? `/attendance?class_id=${classId}&term_id=${termId}&day=${day}` : null);
  const monthly = useLoad<any[]>(`/attendance/monthly?month=${month}`);
  const [marks, setMarks] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [ok, setOk] = useState("");

  useEffect(() => {
    const m: Record<string, string> = {};
    (roster.data || []).forEach((r) => (m[r.student_id] = r.status || "present"));
    setMarks(m);
  }, [roster.data]);

  async function save() {
    setError(""); setOk("");
    try {
      await api("POST", "/attendance", {
        class_id: classId, term_id: termId, day,
        records: Object.entries(marks).map(([student_id, status]) => ({ student_id, status, arrival_time: "", reason: "" })),
      });
      setOk("Attendance saved.");
    } catch (e: any) { setError(e.message); }
  }

  return (
    <>
    <Card title="Take attendance">
      <Msg error={clsError || roster.error || error} ok={ok} />
      <div className="row">
        <Select placeholder="Choose class" value={classId} onChange={(e) => setClassId(e.target.value)} options={options} />
        <input type="date" value={day} onChange={(e) => setDay(e.target.value)} />
      </div>
      {(roster.data || []).map((r) => (
        <div className="row" key={r.student_id} style={{ justifyContent: "space-between" }}>
          <span>{r.last_name} {r.first_name}</span>
          <span className="seg">
            {["present", "late", "absent", "excused"].map((s) => (
              <button key={s} className={marks[r.student_id] === s ? "on" : ""} onClick={() => setMarks({ ...marks, [r.student_id]: s })}>{s}</button>
            ))}
          </span>
        </div>
      ))}
      {roster.data && roster.data.length === 0 && <p className="muted">No students are enrolled in this class for the selected term.</p>}
      {roster.data && roster.data.length > 0 && <button className="primary" onClick={save}>Save attendance</button>}
    </Card>
      <Card title="Monthly attendance report">
        <div className="row"><input type="month" value={month} onChange={(e) => setMonth(e.target.value)} /></div>
        <Msg error={monthly.error} />
        <Table head={["Student","Present","Absent","Late","Excused","Marked","Percentage"]} rows={(monthly.data || []).map((r) => [r.last_name+" "+r.first_name,r.present,r.absent,r.late,r.excused,r.marked,`${r.percentage}%`])} />
      </Card>
    </>
  );
}
