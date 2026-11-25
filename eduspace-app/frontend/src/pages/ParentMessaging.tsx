import { useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, Select, Table, useForm } from "../ui";

export default function ParentMessaging(){
 const guardians=useLoad<any[]>("/guardians"); const messages=useLoad<any[]>("/parent-messages"); const f=useForm({guardian_id:"",subject:"",body:"",channel:"in_app"}); const [error,setError]=useState("");const [ok,setOk]=useState("");
 async function send(){setError("");setOk("");try{await api("POST","/parent-messages",f.v);f.reset();messages.reload();setOk("Message sent to the selected parent.")}catch(e:any){setError(e.message)}}
 return <div className="premium-page"><div className="premium-heading"><div><p>PARENT ENGAGEMENT</p><h1>Individual messaging</h1><span>Send targeted, personal communication to a guardian</span></div></div><Msg error={error||messages.error} ok={ok}/><div className="premium-grid"><Card title="New message"><Select placeholder="Choose parent / guardian" {...f.bind("guardian_id")} options={(guardians.data||[]).map((g:any)=>({id:g.id,label:g.full_name}))}/><input placeholder="Subject" {...f.bind("subject")}/><textarea rows={5} placeholder="Write a clear, personal message" {...f.bind("body")}/><select {...f.bind("channel")}><option value="in_app">In-app message</option><option value="email">Email</option><option value="sms">SMS</option></select><button className="primary" onClick={send}>Send message</button></Card><Card title="Sent messages"><Table head={["Guardian","Subject","Channel","Sent"]} rows={(messages.data||[]).map((m:any)=>[m.guardian_name,m.subject,m.channel,m.sent_at?new Date(m.sent_at).toLocaleString("en-NG",{dateStyle:"medium",timeStyle:"short"}):"—"] )}/></Card></div></div>
}
