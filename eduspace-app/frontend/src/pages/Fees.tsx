import { useState } from "react";
import { api, naira, toKobo, useLoad } from "../api";
import { Card, Msg, Select, Table, useForm } from "../ui";

export default function Fees({ me, termId }: { me: { role: string }; termId: string }) {
  const invoices = useLoad<any[]>("/fees/invoices?term_id=" + termId);
  const classes = useLoad<any[]>(me.role === "bursar" ? null : "/classes");
  const bulk = useForm({ class_id: "", amount: "", description: "School fees" });
  const pay = useForm({ amount: "", method: "cash", reference: "" });
  const [paying, setPaying] = useState("");
  const [receipt, setReceipt] = useState<any>(null);
  const [error, setError] = useState("");
  const [ok, setOk] = useState("");
  const canPay = me.role === "owner" || me.role === "bursar";

  async function bill() {
    setError(""); setOk("");
    try {
      const r = await api<any>("POST", "/fees/invoices/bulk", {
        class_id: bulk.v.class_id, term_id: termId, description: bulk.v.description, amount_kobo: toKobo(bulk.v.amount) });
      setOk(`${r.invoices_created} invoices created.`); invoices.reload();
    } catch (e: any) { setError(e.message); }
  }
  async function record(inv: any) {
    setError(""); setOk("");
    try {
      const r = await api<any>("POST", "/fees/payments", {
        invoice_id: inv.id, amount_kobo: toKobo(pay.v.amount), method: pay.v.method, reference: pay.v.reference });
      setReceipt({ student: `${inv.first_name} ${inv.last_name}`, what: inv.description, amount: toKobo(pay.v.amount),
        method: pay.v.method, balance: r.balance_kobo, id: r.id, date: new Date().toLocaleString() });
      setPaying(""); pay.reset(); invoices.reload();
    } catch (e: any) { setError(e.message); }
  }

  if (receipt) return (
    <div className="paper">
      <h2>Payment receipt</h2>
      <p>Receipt no. {receipt.id.slice(0, 8)} · {receipt.date}</p>
      <p><b>{receipt.student}</b> — {receipt.what}</p>
      <p>Amount paid: <b>{naira(receipt.amount)}</b> ({receipt.method})</p>
      <p>Balance remaining: <b>{naira(receipt.balance)}</b></p>
      <div className="noprint row">
        <button className="primary" onClick={() => window.print()}>Print receipt</button>
        <button className="small ghost" onClick={() => setReceipt(null)}>Close</button>
      </div>
    </div>
  );

  return (
    <>
      <Msg error={error || invoices.error} ok={ok} />
      {me.role !== "bursar" && (
        <Card title="Bill a whole class for the selected term">
          <div className="row">
            <Select placeholder="Class" {...bulk.bind("class_id")} options={(classes.data || []).map((c) => ({ id: c.id, label: c.name }))} />
            <input type="number" placeholder="Amount per student (₦)" {...bulk.bind("amount")} />
            <input placeholder="Description" {...bulk.bind("description")} />
          </div>
          <button className="primary" disabled={!termId} onClick={bill}>Create invoices</button>
        </Card>
      )}
      <Card title="Invoices">
        <Table
          head={["Student", "Fee", "Amount", "Paid", "Balance", "Status", ""]}
          rows={(invoices.data || []).map((i) => [
            `${i.last_name} ${i.first_name}`, i.description, naira(i.amount_kobo), naira(i.paid_kobo), naira(i.balance_kobo), i.status,
            canPay && i.balance_kobo > 0 ? (
              paying === i.id ? (
                <span className="row" style={{ margin: 0 }}>
                  <input type="number" placeholder="₦" style={{ flex: "0 0 90px" }} {...pay.bind("amount")} />
                  <select style={{ flex: "0 0 90px" }} {...pay.bind("method")}>
                    <option value="cash">Cash</option><option value="transfer">Transfer</option><option value="pos">POS</option>
                  </select>
                  <button className="small" onClick={() => record(i)}>Save</button>
                </span>
              ) : <button className="small" onClick={() => setPaying(i.id)}>Record payment</button>
            ) : "",
          ])}
        />
      </Card>
    </>
  );
}
