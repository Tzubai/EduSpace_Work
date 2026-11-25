"""End-to-end check of EduSpace against a running server.
Usage: python3 scripts/smoke_test.py http://localhost:8080
Uses fake data only. Run it against a development database."""
import json, sys, time, urllib.request, urllib.error

BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080").rstrip("/") + "/api/v1"
RUN = str(int(time.time()))
passed = failed = 0

def call(method, path, body=None, token=None):
    req = urllib.request.Request(BASE + path, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Content-Type": "application/json", **({"Authorization": "Bearer " + token} if token else {})})
    try:
        with urllib.request.urlopen(req) as r:
            return r.status, json.loads(r.read() or "null")
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read() or "null")

def check(name, got, want):
    global passed, failed
    ok = got == want
    passed += ok; failed += (not ok)
    print(("PASS " if ok else "FAIL ") + name + ("" if ok else f"  (got {got}, wanted {want})"))

def reg(name):
    s, r = call("POST", "/auth/register-institute", {"institute_name": name, "full_name": name + " Owner",
        "email": f"owner{RUN}@{name.lower().replace(' ','')}.test", "password": "password123"})
    check("register " + name, s, 201); return r["token"]

A, B = reg("Alpha School"), reg("Beta School")

_, cls = call("POST", "/classes", {"name": "JSS 1"}, A)
_, sub = call("POST", "/subjects", {"name": "Mathematics"}, A)
_, ses = call("POST", "/sessions", {"name": "2026/2027"}, A)
s, term = call("POST", "/terms", {"session_id": ses["id"], "name": "First Term", "starts_on": "2026-09-14", "ends_on": "2026-12-18"}, A)
check("create term", s, 201)
_, s1 = call("POST", "/students", {"first_name": "Amina", "last_name": "Yusuf", "admission_number": "001", "date_of_birth": "2015-03-02"}, A)
_, s2 = call("POST", "/students", {"first_name": "Bola", "last_name": "Adeyemi", "admission_number": "002"}, A)
check("duplicate admission number rejected", call("POST", "/students", {"first_name": "X", "last_name": "Y", "admission_number": "001"}, A)[0], 409)
for st in (s1, s2):
    check("enroll student", call("POST", "/enrollments", {"student_id": st["id"], "class_id": cls["id"], "term_id": term["id"]}, A)[0], 201)

s, t = call("POST", "/users", {"full_name": "Mr Musa", "email": f"musa{RUN}@a.test", "password": "password123", "role": "teacher"}, A)
check("create teacher", s, 201)
_, tl = call("POST", "/auth/login", {"email": f"musa{RUN}@a.test", "password": "password123"})
T = tl["token"]
check("unassigned teacher blocked from attendance", call("GET", f"/attendance?class_id={cls['id']}&term_id={term['id']}&day=2026-11-18", token=T)[0], 403)
check("assign teacher", call("POST", "/teacher-assignments", {"user_id": t["id"], "class_id": cls["id"], "subject_id": sub["id"]}, A)[0], 201)
check("teacher takes attendance", call("POST", "/attendance", {"class_id": cls["id"], "term_id": term["id"], "day": "2026-11-18",
    "records": [{"student_id": s1["id"], "status": "present"}, {"student_id": s2["id"], "status": "absent"}]}, T)[0], 200)
check("bad attendance status rejected", call("POST", "/attendance", {"class_id": cls["id"], "term_id": term["id"], "day": "2026-11-18",
    "records": [{"student_id": s1["id"], "status": "maybe"}]}, T)[0], 400)

_, asm = call("POST", "/results/assessments", {"class_id": cls["id"], "subject_id": sub["id"], "term_id": term["id"], "title": "Test 1", "max_score": 40}, T)
check("score above max rejected", call("POST", "/results/grades", {"assessment_id": asm["id"], "scores": [{"student_id": s1["id"], "score": 55}]}, T)[0], 400)
check("teacher enters scores", call("POST", "/results/grades", {"assessment_id": asm["id"],
    "scores": [{"student_id": s1["id"], "score": 36}, {"student_id": s2["id"], "score": 18}]}, T)[0], 200)

check("bulk invoices", call("POST", "/fees/invoices/bulk", {"class_id": cls["id"], "term_id": term["id"], "amount_kobo": 3500000}, A)[1]["invoices_created"], 2)
_, inv = call("GET", f"/fees/invoices?term_id={term['id']}", token=A)
i1 = next(i for i in inv if i["admission_number"] == "001")
check("teacher cannot record payment", call("POST", "/fees/payments", {"invoice_id": i1["id"], "amount_kobo": 100, "method": "cash"}, T)[0], 403)
s, p = call("POST", "/fees/payments", {"invoice_id": i1["id"], "amount_kobo": 2000000, "method": "transfer", "reference": "TRF-1"}, A)
check("record part payment", (s, p["balance_kobo"]), (201, 1500000))
check("overpayment rejected", call("POST", "/fees/payments", {"invoice_id": i1["id"], "amount_kobo": 1500001, "method": "cash"}, A)[0], 400)
check("final payment", call("POST", "/fees/payments", {"invoice_id": i1["id"], "amount_kobo": 1500000, "method": "cash"}, A)[1]["balance_kobo"], 0)

s, g = call("POST", "/guardians", {"full_name": "Mrs Yusuf", "phone": "08000000000", "email": f"mum{RUN}@a.test",
    "password": "password123", "student_ids": [s1["id"]]}, A)
check("create parent", s, 201)
P = call("POST", "/auth/login", {"email": f"mum{RUN}@a.test", "password": "password123"})[1]["token"]
_, kids = call("GET", "/parent/children", token=P)
check("parent sees only own child", [k["first_name"] for k in kids], ["Amina"])
s, rc = call("GET", f"/results/report-card/{s1['id']}?term_id={term['id']}", token=P)
check("parent report card", (s, rc["subjects"][0]["percent"], rc["subjects"][0]["grade"]), (200, 90.0, "A"))
check("parent blocked from other child's report card", call("GET", f"/results/report-card/{s2['id']}?term_id={term['id']}", token=P)[0], 403)
check("parent blocked from student list", call("GET", "/students", token=P)[0], 403)
check("announcement", call("POST", "/announcements", {"title": "Resumption", "body": "School resumes Monday."}, A)[0], 201)
check("parent reads announcements", len(call("GET", "/announcements", token=P)[1]), 1)
d = call("GET", "/dashboard", token=A)[1]
check("dashboard", (d["students"], d["fees"]["billed_kobo"], d["fees"]["collected_kobo"]), (2, 7000000, 3500000))

print("--- school isolation ---")
check("Beta sees no Alpha students", call("GET", "/students", token=B)[1], [])
check("Beta cannot read Alpha student by id", call("GET", f"/students/{s1['id']}", token=B)[1], [])
check("Beta cannot edit Alpha student", call("PUT", f"/students/{s1['id']}", {"first_name": "Hacked", "last_name": "X", "admission_number": "9"}, B)[0], 404)
check("Beta cannot delete Alpha student", call("DELETE", f"/students/{s1['id']}", token=B)[0], 404)
check("Beta cannot enroll Alpha student", call("POST", "/enrollments", {"student_id": s1["id"], "class_id": cls["id"], "term_id": term["id"]}, B)[0], 400)
check("Beta cannot pay Alpha invoice", call("POST", "/fees/payments", {"invoice_id": i1["id"], "amount_kobo": 100, "method": "cash"}, B)[0], 404)
check("Beta cannot see Alpha report card", call("GET", f"/results/report-card/{s1['id']}?term_id={term['id']}", token=B)[0], 404)
check("Beta cannot grade Alpha assessment", call("POST", "/results/grades", {"assessment_id": asm["id"], "scores": [{"student_id": s1["id"], "score": 1}]}, B)[0], 404)
check("Beta sees no Alpha invoices", call("GET", "/fees/invoices", token=B)[1], [])
check("Beta sees no Alpha announcements", call("GET", "/announcements", token=B)[1], [])
check("no token rejected", call("GET", "/students")[0], 401)
check("forged token rejected", call("GET", "/students", token=A[:-3] + "xyz")[0], 401)
check("wrong password rejected", call("POST", "/auth/login", {"email": f"mum{RUN}@a.test", "password": "wrongpass1"})[0], 401)

print(f"\n{passed} passed, {failed} failed")
sys.exit(1 if failed else 0)
