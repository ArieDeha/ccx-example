# ccx-example — Video Publish (ccx + ccxpolicy)

A small example showing how to orchestrate a 3+ level workflow with
**ccx** (cascading context) and **ccxpolicy** (separate policy engine).

## Run
```bash
go run ./cmd/video_publish

```

---
## File Structure
```text
ccx-example/
├─ go.mod
│  └─ go.sum
├─ README.md
├─ cmd/
│  └─ video_publish/
│     └─ main.go
└─ policies/
   ├─ quality_cap.go
   └─ safety_stop.go

```

asd