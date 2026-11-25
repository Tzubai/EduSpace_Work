import { useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, Table, useForm } from "../ui";
import ReportCard from "./ReportCard";

export default function Students({ termId }: { termId: string }) {
  const list = useLoad<any[]>("/students");
  const f = useForm({ first_name: "", last_name: "", admission_number: "", date_of_birth: "" });
  const [error, setError] = useState("");
  const [card, setCard] = useState("");

  async function add() {
    setError("");
    try {
      await api("POST", "/students", f.v);
      f.reset();
      list.reload();
    } catch (e: any) { setError(e.message); }
  }
  async function remove(id: string) {
    if (!confirm("Delete this student? This cannot be undone.")) return;
    try { await api("DELETE", "/students/" + id); list.reload(); } catch (e: any) { setError(e.message); }
  }

  if (card) return <ReportCard studentId={card} termId={termId} onClose={() => setCard("")} />;
  return (
    <>
      <Card title="Add student">
        <div className="row">
          <input placeholder="First name" {...f.bind("first_name")} />
          <input placeholder="Last name" {...f.bind("last_name")} />
          <input placeholder="Admission number" {...f.bind("admission_number")} />
          <input type="date" title="Date of birth" {...f.bind("date_of_birth")} />
        </div>
        <Msg error={error} />
        <button className="primary" onClick={add}>Add student</button>
      </Card>
      <Card title={`Students (${list.data?.length ?? 0})`}>
        <Msg error={list.error} />
        <Table
          head={["Admission no.", "Name", "Status", ""]}
          rows={(list.data || []).map((s) => [
            s.admission_number, `${s.last_name} ${s.first_name}`, s.status,
            <span className="row" style={{ margin: 0 }}>
              <button className="small" disabled={!termId} onClick={() => setCard(s.id)}>Report card</button>
              <button className="small danger" onClick={() => remove(s.id)}>Delete</button>
            </span>,
          ])}
        />
      </Card>
    </>
  );
}
