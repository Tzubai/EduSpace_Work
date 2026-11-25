import { useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, Select, Table, useForm } from "../ui";

export default function People({ me }: { me: { role: string } }) {
  const users = useLoad<any[]>("/users");
  const guardians = useLoad<any[]>("/guardians");
  const students = useLoad<any[]>("/students");
  const u = useForm({ full_name: "", email: "", password: "", role: "teacher" });
  const g = useForm({ full_name: "", phone: "", email: "", password: "", student: "" });
  const [error, setError] = useState("");
  const [ok, setOk] = useState("");

  async function run(fn: () => Promise<any>, after: () => void) {
    setError(""); setOk("");
    try { await fn(); setOk("Saved."); after(); } catch (e: any) { setError(e.message); }
  }

  return (
    <>
      <Msg error={error} ok={ok} />
      <Card title="Add staff login (teacher, bursar, admin)">
        <div className="row">
          <input placeholder="Full name" {...u.bind("full_name")} />
          <input type="email" placeholder="Email" {...u.bind("email")} />
          <input type="password" placeholder="Temporary password (8+)" {...u.bind("password")} />
          <select {...u.bind("role")}>
            <option value="teacher">Teacher</option>
            <option value="bursar">Bursar</option>
            {me.role === "owner" && <option value="admin">Administrator</option>}
          </select>
        </div>
        <button className="primary" onClick={() => run(() => api("POST", "/users", u.v), () => { u.reset(); users.reload(); })}>Add</button>
        <Table head={["Name", "Email", "Role", "Status", ""]} rows={(users.data || []).map((x) => [
          x.full_name,
          x.email,
          x.role,
          x.is_active === false ? "Inactive" : "Active",
          x.role === "owner" ? "Protected" : <button className="small ghost" onClick={() => run(
            () => api("PUT", `/users/${x.id}`, { is_active: x.is_active === false }),
            () => users.reload())}>{x.is_active === false ? "Activate" : "Deactivate"}</button>
        ])} />
      </Card>
      <Card title="Add parent / guardian">
        <div className="row">
          <input placeholder="Full name" {...g.bind("full_name")} />
          <input placeholder="Phone" {...g.bind("phone")} />
          <Select placeholder="Child" {...g.bind("student")}
            options={(students.data || []).map((s) => ({ id: s.id, label: `${s.last_name} ${s.first_name}` }))} />
        </div>
        <div className="row">
          <input type="email" placeholder="Email for parent login (optional)" {...g.bind("email")} />
          <input type="password" placeholder="Password (8+, if email given)" {...g.bind("password")} />
        </div>
        <button className="primary" onClick={() => run(
          () => api("POST", "/guardians", { full_name: g.v.full_name, phone: g.v.phone, email: g.v.email, password: g.v.password,
            student_ids: g.v.student ? [g.v.student] : [] }),
          () => { g.reset(); guardians.reload(); })}>Add parent</button>
        <Table head={["Name", "Phone", "Children"]} rows={(guardians.data || []).map((x) => [x.full_name, x.phone, x.children])} />
      </Card>
    </>
  );
}
