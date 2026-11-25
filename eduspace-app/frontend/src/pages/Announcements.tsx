import { useState } from "react";
import { api, useLoad } from "../api";
import { Card, Msg, useForm } from "../ui";

export default function Announcements({ me }: { me: { role: string } }) {
  const list = useLoad<any[]>("/announcements");
  const f = useForm({ title: "", body: "" });
  const [error, setError] = useState("");
  const canPost = me.role === "owner" || me.role === "admin";

  async function post() {
    setError("");
    try { await api("POST", "/announcements", f.v); f.reset(); list.reload(); } catch (e: any) { setError(e.message); }
  }
  return (
    <>
      {canPost && (
        <Card title="Post an announcement">
          <div className="row">
            <input placeholder="Title" {...f.bind("title")} />
          </div>
          <div className="row">
            <textarea rows={3} placeholder="Message to parents and staff" {...f.bind("body")} />
          </div>
          <Msg error={error} />
          <button className="primary" onClick={post}>Post</button>
        </Card>
      )}
      <Card title="Announcements">
        <Msg error={list.error} />
        {(list.data || []).length === 0 && <p className="muted">No announcements yet.</p>}
        {(list.data || []).map((a) => (
          <div key={a.id} style={{ marginBottom: 12 }}>
            <b>{a.title}</b> <span className="muted">· {new Date(a.created_at).toLocaleDateString()}</span>
            <div>{a.body}</div>
          </div>
        ))}
      </Card>
    </>
  );
}
