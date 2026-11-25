import { naira, useLoad } from "../api";
import { Msg } from "../ui";

type DashboardData = {
  students: number;
  teachers: number;
  classes: number;
  attendance_today: { present: number; marked: number };
  fees: { billed_kobo: number; collected_kobo: number; outstanding_kobo: number };
};

type Student = {
  id: string;
  first_name: string;
  last_name: string;
  admission_number: string;
  status: string;
};

type Props = {
  termId: string;
  me: { full_name: string; institute_name: string; role: string };
  onNavigate: (page: string) => void;
};

const shortDate = new Intl.DateTimeFormat("en-NG", {
  weekday: "long", day: "numeric", month: "long", year: "numeric",
}).format(new Date());

export default function Dashboard({ termId, me, onNavigate }: Props) {
  const { data: d, error, reload } = useLoad<DashboardData>("/dashboard?term_id=" + termId);
  const students = useLoad<Student[]>("/students");
  const announcements = useLoad<any[]>("/announcements");

  if (error) return <div className="dashboard-wrap"><Msg error={error} /><button className="small ghost" onClick={reload}>Try again</button></div>;
  if (!d) return <div className="dashboard-wrap"><div className="dashboard-loading">Loading your school overview…</div></div>;

  const attendanceRate = d.attendance_today.marked
    ? Math.round((d.attendance_today.present / d.attendance_today.marked) * 100)
    : 0;
  const collectionRate = d.fees.billed_kobo
    ? Math.min(100, Math.round((d.fees.collected_kobo / d.fees.billed_kobo) * 100))
    : 0;
  const recentStudents = [...(students.data || [])].slice(0, 5);
  const recentAnnouncements = [...(announcements.data || [])].slice(0, 3);
  const greeting = new Date().getHours() < 12 ? "Good morning" : new Date().getHours() < 17 ? "Good afternoon" : "Good evening";

  const quickActions = [
    { icon: "＋", title: "Register a student", detail: "Add a new student record", page: "Students" },
    { icon: "♙", title: "Manage staff", detail: "Teachers and school personnel", page: "People" },
    { icon: "▦", title: "Set up classes", detail: "Classes, subjects and terms", page: "Setup" },
    { icon: "₦", title: "Manage school fees", detail: "Invoices and payment records", page: "Fees" },
  ];

  return (
    <div className="dashboard-wrap">
      <section className="dashboard-welcome">
        <div>
          <p className="dashboard-kicker">SCHOOL OVERVIEW</p>
          <h2>{greeting}, {me.full_name.split(" ")[0]} <span>👋</span></h2>
          <p>Here’s what is happening at <strong>{me.institute_name}</strong> today.</p>
        </div>
        <div className="dashboard-date">
          <span className="date-icon">▦</span>
          <span>{shortDate}</span>
        </div>
      </section>

      <section className="dashboard-section">
        <div className="dashboard-section-heading">
          <div><h3>At a glance</h3><p>Your school's key figures for the selected term.</p></div>
          <button className="small ghost" onClick={() => { reload(); students.reload(); announcements.reload(); }}>↻ Refresh</button>
        </div>
        <div className="dashboard-metrics dashboard-metrics-eight">
          <article className="dashboard-metric pastel-mint"><div className="metric-icon teal">♙</div><span className="metric-label">Active students</span><strong>{d.students.toLocaleString("en-NG")}</strong><small>Currently enrolled records</small></article>
          <article className="dashboard-metric pastel-blue"><div className="metric-icon blue">♟</div><span className="metric-label">Teachers</span><strong>{d.teachers.toLocaleString("en-NG")}</strong><small>Registered teacher accounts</small></article>
          <article className="dashboard-metric pastel-lilac"><div className="metric-icon violet">▦</div><span className="metric-label">Classes</span><strong>{d.classes.toLocaleString("en-NG")}</strong><small>School classes created</small></article>
          <article className="dashboard-metric pastel-peach"><div className="metric-icon amber">✓</div><span className="metric-label">Present today</span><strong>{d.attendance_today.present}<em> / {d.attendance_today.marked}</em></strong><small>Attendance marked</small></article>
          <article className="dashboard-metric pastel-yellow"><div className="metric-icon amber">◉</div><span className="metric-label">Attendance rate</span><strong>{attendanceRate}%</strong><small>Of marked records</small></article>
          <article className="dashboard-metric pastel-rose"><div className="metric-icon violet">₦</div><span className="metric-label">Fees billed</span><strong className="metric-money">{naira(d.fees.billed_kobo)}</strong><small>Current term total</small></article>
          <article className="dashboard-metric pastel-mint"><div className="metric-icon teal">↗</div><span className="metric-label">Fees collected</span><strong className="metric-money">{naira(d.fees.collected_kobo)}</strong><small>Payments recorded</small></article>
          <article className="dashboard-metric pastel-blue"><div className="metric-icon blue">◷</div><span className="metric-label">Outstanding fees</span><strong className="metric-money">{naira(d.fees.outstanding_kobo)}</strong><small>Awaiting payment</small></article>
        </div>
      </section>

      <section className="dashboard-insights">
        <article className="dashboard-panel finance-panel">
          <div className="panel-heading"><div><h3>Fee collection</h3><p>Payment summary for the selected term</p></div><span className="panel-symbol">₦</span></div>
          <div className="finance-total">{naira(d.fees.collected_kobo)}<small> collected</small></div>
          <div className="progress-track"><span style={{ width: `${collectionRate}%` }} /></div>
          <div className="progress-caption"><span>{collectionRate}% collected</span><span>{naira(d.fees.billed_kobo)} billed</span></div>
          <div className="finance-breakdown">
            <div><span className="legend-dot collected" />Collected<strong>{naira(d.fees.collected_kobo)}</strong></div>
            <div><span className="legend-dot outstanding" />Outstanding<strong>{naira(d.fees.outstanding_kobo)}</strong></div>
          </div>
          <button className="text-link" onClick={() => onNavigate("Fees")}>View fee records <span>→</span></button>
        </article>
        <article className="dashboard-panel attendance-panel">
          <div className="panel-heading"><div><h3>Today's attendance</h3><p>Attendance records entered today</p></div><span className="panel-symbol green">✓</span></div>
          <div className="attendance-visual">
            <div className="attendance-ring" style={{ background: `conic-gradient(#168f89 ${attendanceRate * 3.6}deg, #e9eef3 0deg)` }}>
              <div><strong>{attendanceRate}%</strong><small>present</small></div>
            </div>
            <div className="attendance-legend">
              <div><span className="legend-dot collected" />Present / late<strong>{d.attendance_today.present}</strong></div>
              <div><span className="legend-dot outstanding" />Marked records<strong>{d.attendance_today.marked}</strong></div>
              <small>{d.attendance_today.marked ? "Based on records entered today." : "Mark attendance to see today's summary."}</small>
            </div>
          </div>
          <button className="text-link" onClick={() => onNavigate("Attendance")}>Open attendance <span>→</span></button>
        </article>
      </section>

      <section className="dashboard-section">
        <div className="dashboard-section-heading"><div><h3>Quick actions</h3><p>Common tasks to help you run your school.</p></div></div>
        <div className="quick-actions">
          {quickActions.map((action) => (
            <button className="quick-action" key={action.title} onClick={() => onNavigate(action.page)}>
              <span className="quick-action-icon">{action.icon}</span>
              <span><strong>{action.title}</strong><small>{action.detail}</small></span>
              <b>→</b>
            </button>
          ))}
        </div>
      </section>

      <section className="dashboard-bottom">
        <article className="dashboard-panel">
          <div className="panel-heading"><div><h3>Student records</h3><p>A quick look at your student directory</p></div><button className="text-link" onClick={() => onNavigate("Students")}>View all →</button></div>
          {students.error ? <Msg error={students.error} /> : recentStudents.length ? (
            <div className="recent-list">{recentStudents.map((student) => (
              <div className="recent-row" key={student.id}>
                <span className="student-avatar">{`${student.first_name[0] || ""}${student.last_name[0] || ""}`.toUpperCase()}</span>
                <span className="recent-student-name"><strong>{student.last_name} {student.first_name}</strong><small>{student.admission_number}</small></span>
                <span className={`status-pill ${student.status === "active" ? "active" : ""}`}>{student.status}</span>
              </div>
            ))}</div>
          ) : <div className="dashboard-empty">No student records yet. Add your first student to get started.</div>}
        </article>
        <article className="dashboard-panel">
          <div className="panel-heading"><div><h3>School announcements</h3><p>Latest messages for your school community</p></div><button className="text-link" onClick={() => onNavigate("Announcements")}>View all →</button></div>
          {announcements.error ? <Msg error={announcements.error} /> : recentAnnouncements.length ? (
            <div className="announcement-list">{recentAnnouncements.map((item, i) => (
              <div className="announcement-row" key={item.id || i}><span className="announcement-bullet">•</span><div><strong>{item.title}</strong><small>{item.created_at ? new Date(item.created_at).toLocaleDateString("en-NG", { day: "numeric", month: "short", year: "numeric" }) : "School update"}</small></div></div>
            ))}</div>
          ) : <div className="dashboard-empty">No announcements yet. Share an update with your school community.</div>}
        </article>
      </section>
      <p className="dashboard-footnote">Dashboard figures are based on records currently saved in EduSpace.</p>
    </div>
  );
}
