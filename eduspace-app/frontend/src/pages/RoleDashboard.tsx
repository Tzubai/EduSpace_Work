import { naira, useLoad } from "../api";
import { Msg } from "../ui";

type Assignment = { id: string; class: string; subject: string; class_id: string; subject_id?: string };
type Child = { student_id: string; first_name: string; last_name: string; days_present: number; days_absent: number; billed_kobo: number; paid_kobo: number };
type Props = { me: { full_name: string; institute_name: string; role: string }; termId: string; onNavigate: (page: string) => void };
const today = new Intl.DateTimeFormat("en-NG", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date());

export default function RoleDashboard({ me, termId, onNavigate }: Props) {
  const teacher = me.role === "teacher";
  const assignments = useLoad<Assignment[]>(teacher ? "/teacher-assignments" : null);
  const children = useLoad<Child[]>(me.role === "parent" ? "/parent/children" : null);
  const greeting = new Date().getHours() < 12 ? "Good morning" : new Date().getHours() < 17 ? "Good afternoon" : "Good evening";
  if (teacher) {
    const list = Array.isArray(assignments.data) ? assignments.data : [];
    return <div className="role-dashboard">
      <section className="dashboard-welcome"><div><p className="dashboard-kicker">TEACHER WORKSPACE</p><h2>{greeting}, {me.full_name.split(" ")[0]} 👋</h2><p>Here is your teaching overview for <strong>{me.institute_name}</strong>.</p></div><div className="dashboard-date">▦ &nbsp;{today}</div></section>
      <Msg error={assignments.error} />
      <section className="role-stat-grid"><article className="role-stat"><span>Assigned classes</span><strong>{assignments.data?.length ?? "—"}</strong><small>Your class assignments</small></article><article className="role-stat"><span>Term workspace</span><strong>{termId ? "Ready" : "Not set"}</strong><small>Selected academic term</small></article></section>
      <section className="dashboard-section"><div className="dashboard-section-heading"><div><h3>My teaching assignments</h3><p>Classes and subjects assigned to your account.</p></div><button className="small ghost" onClick={assignments.reload}>↻ Refresh</button></div>
      {assignments.data === null && !assignments.error ? <div className="dashboard-loading">Loading assignments…</div> : list.length === 0 ? <div className="role-empty">No teaching assignments yet. Ask your school administrator to assign your classes.</div> : <div className="role-assignment-grid">{list.map((a) => <article className="role-assignment" key={a.id}><div className="role-assignment-icon">▤</div><strong>{a.class}</strong><span>{a.subject}</span><div className="role-assignment-actions"><button onClick={() => onNavigate("Attendance")}>Take attendance</button><button onClick={() => onNavigate("Scores")}>Manage scores</button></div></article>)}</div>}</section>
      <section className="role-action-row"><button onClick={() => onNavigate("Attendance")}>✓ <span><strong>Attendance</strong><small>Mark class attendance</small></span> →</button><button onClick={() => onNavigate("Scores")}>▤ <span><strong>Assessments & scores</strong><small>Record academic performance</small></span> →</button><button onClick={() => onNavigate("Announcements")}>✉ <span><strong>Announcements</strong><small>School updates</small></span> →</button></section>
    </div>;
  }
  const kidList = Array.isArray(children.data) ? children.data : [];
  const present = kidList.reduce((n, k) => n + Number(k.days_present || 0), 0);
  const absent = kidList.reduce((n, k) => n + Number(k.days_absent || 0), 0);
  const balance = kidList.reduce((n, k) => n + Number(k.billed_kobo || 0) - Number(k.paid_kobo || 0), 0);
  return <div className="role-dashboard"><section className="dashboard-welcome"><div><p className="dashboard-kicker">FAMILY WORKSPACE</p><h2>{greeting}, {me.full_name.split(" ")[0]} 👋</h2><p>Stay connected with your children's school journey at <strong>{me.institute_name}</strong>.</p></div><div className="dashboard-date">▦ &nbsp;{today}</div></section>
    <Msg error={children.error} />
    <section className="role-stat-grid"><article className="role-stat"><span>Linked children</span><strong>{children.data?.length ?? "—"}</strong><small>Connected student records</small></article><article className="role-stat"><span>Attendance days present</span><strong>{children.data ? present : "—"}</strong><small>Across linked children</small></article><article className="role-stat"><span>Days absent</span><strong>{children.data ? absent : "—"}</strong><small>Across linked children</small></article><article className="role-stat"><span>Fees balance</span><strong className="role-stat-money">{children.data ? naira(balance) : "—"}</strong><small>Combined outstanding amount</small></article></section>
    <section className="dashboard-section"><div className="dashboard-section-heading"><div><h3>Your children</h3><p>Quick access to academic and school information.</p></div><button className="small ghost" onClick={children.reload}>↻ Refresh</button></div>{children.data === null && !children.error ? <div className="dashboard-loading">Loading linked student information…</div> : kidList.length === 0 ? <div className="role-empty">No children are linked to your account yet. Contact the school to link your account.</div> : <div className="role-child-grid">{kidList.map((k) => <article className="role-child" key={k.student_id}><div className="role-child-avatar">{`${k.first_name[0] || ""}${k.last_name[0] || ""}`.toUpperCase()}</div><div className="role-child-main"><strong>{k.first_name} {k.last_name}</strong><small>Attendance: {k.days_present} present · {k.days_absent} absent</small><small>Fee balance: <b>{naira(k.billed_kobo-k.paid_kobo)}</b></small></div><button onClick={() => onNavigate("My children")}>View details →</button></article>)}</div>}</section>
    <section className="role-action-row"><button onClick={() => onNavigate("My children")}>♡ <span><strong>Children's records</strong><small>Attendance, fees, and report cards</small></span> →</button><button onClick={() => onNavigate("Announcements")}>✉ <span><strong>Announcements</strong><small>Important school notices</small></span> →</button></section>
  </div>;
}
