import { useState } from "react";
import { api, setToken } from "../api";
import { Card, Msg, useForm } from "../ui";

export default function Auth({ onDone, onBack, initialMode = "login" }: { onDone: () => void; onBack: () => void; initialMode?: "login" | "register" }) {
  const [mode, setMode] = useState<"login" | "register">(initialMode);
  const [error, setError] = useState("");
  const f = useForm({ institute_name: "", full_name: "", email: "", password: "" });

  async function submit() {
    setError("");
    try {
      const body = mode === "login" ? { email: f.v.email, password: f.v.password } : f.v;
      const r = await api<{ token: string }>("POST", mode === "login" ? "/auth/login" : "/auth/register-institute", body);
      setToken(r.token);
      onDone();
    } catch (e: any) {
      setError(e.message);
    }
  }

  return (
    <main className="auth">
      <button className="text-button auth-back" onClick={onBack}>← Back to EduSpace</button>
      <img className="auth-logo" src="/eduspace-logo.png" alt="EduSpace" />
      <Card title={mode === "login" ? "Log in" : "Register your school"}>
        {mode === "register" && (
          <div className="row">
            <input placeholder="School name" {...f.bind("institute_name")} />
            <input placeholder="Your full name" {...f.bind("full_name")} />
          </div>
        )}
        <div className="row">
          <input type="email" placeholder="Email" {...f.bind("email")} />
          <input type="password" placeholder="Password (8+ characters)" {...f.bind("password")} />
        </div>
        <Msg error={error} />
        <div className="row">
          <button className="primary" onClick={submit}>{mode === "login" ? "Log in" : "Create school"}</button>
          <button className="small ghost" onClick={() => setMode(mode === "login" ? "register" : "login")}>
            {mode === "login" ? "New school? Register" : "Have an account? Log in"}
          </button>
        </div>
      </Card>
    </main>
  );
}
