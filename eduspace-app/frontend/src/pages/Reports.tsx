import { naira, useLoad } from "../api";
import { Msg } from "../ui";

type Data={students:number;inactive_students:number;teachers:number;staff:number;classes:number;enrollments:number;attendance_today:{present:number;marked:number;rate:number};fees:{billed_kobo:number;collected_kobo:number;outstanding_kobo:number};academic:{average_percent:number};alerts:any[]};

export default function Reports({termId}:{termId:string}){
 const d=useLoad<Data>("/dashboard?term_id="+termId);
 if(d.error)return <Msg error={d.error}/>;
 if(!d.data)return <div className="dashboard-loading">Preparing reports…</div>;
 const x=d.data;
 const collection=x.fees.billed_kobo?Math.round(x.fees.collected_kobo/x.fees.billed_kobo*100):0;
 return <div className="premium-page">
  <div className="premium-heading"><div><p>REPORTS & ANALYTICS</p><h1>School intelligence</h1><span>Live operational summaries for the selected term</span></div><button className="small ghost" onClick={()=>window.print()}>Print report</button></div>
  <div className="report-grid">
   <article><span>Enrollment</span><strong>{x.enrollments}</strong><small>{x.students} active student records</small></article>
   <article><span>Attendance</span><strong>{x.attendance_today.rate}%</strong><small>{x.attendance_today.present} present today</small></article>
   <article><span>Fee collection</span><strong>{collection}%</strong><small>{naira(x.fees.collected_kobo)} collected</small></article>
   <article><span>Outstanding</span><strong>{naira(x.fees.outstanding_kobo)}</strong><small>Awaiting collection</small></article>
   <article><span>Academic average</span><strong>{Math.round(x.academic.average_percent)}%</strong><small>Recorded assessments</small></article>
   <article><span>Staff</span><strong>{x.staff}</strong><small>{x.teachers} teachers</small></article>
  </div>
  <section className="report-panel"><h2>Administrative interpretation</h2><div className="report-lines">
   <p><b>Enrollment health:</b> {x.enrollments} students are enrolled for the selected term.</p>
   <p><b>Attendance risk:</b> {x.attendance_today.rate<85?"Attention required — daily attendance is below 85%.":"Attendance is within the target range."}</p>
   <p><b>Finance risk:</b> {x.fees.outstanding_kobo>0?`${naira(x.fees.outstanding_kobo)} remains outstanding.`:"No outstanding fees in the selected term."}</p>
   <p><b>Academic health:</b> {x.academic.average_percent>=50?"Overall recorded performance is satisfactory.":"Overall academic performance needs review."}</p>
  </div></section>
 </div>;
}
