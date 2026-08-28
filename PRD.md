# PRD: EnvMan — CLI Environment Variable Sync Checker

**Versi:** 0.1 (Draft) — implementasi: v0.2.0
**Pemilik Produk:** Novian
**Tanggal:** 28 Agustus 2026
**Status:** Draft untuk pengembangan — Fase 1 & 2 selesai diimplementasikan, kode belum ter-push ke main (nunggu push protection resolution)

---

## 1. Latar Belakang & Masalah

Saat mengelola aplikasi yang di-deploy ke VPS (Docker Compose, Next.js, dsb), environment variable sering jadi sumber bug yang sulit dilacak:

- `.env.example` di repo tidak sinkron dengan `.env` yang sebenarnya dipakai di lokal maupun production.
- Variable baru ditambahkan developer tapi lupa di-set di server → aplikasi crash saat deploy.
- Tidak ada cara cepat untuk tahu apakah environment lokal, staging, dan production punya konfigurasi yang konsisten.
- Secret asli kadang tidak sengaja ikut ter-commit di file contoh.

Saat ini proses pengecekan dilakukan manual (buka file satu-satu, bandingkan dengan mata), yang lambat dan rawan human error — terutama saat deploy manual ke VPS budget tanpa secret manager canggih.

## 2. Tujuan Produk

Membuat CLI tool (`envman`) yang membantu developer/DevOps memvalidasi dan menyinkronkan environment variable antar file lokal, docker-compose, dan server remote — sehingga masalah env terdeteksi **sebelum** deploy, bukan setelah aplikasi down.

### Tujuan spesifik
- Mengurangi insiden "app crash karena env variable hilang" saat deploy.
- Memberi visibilitas cepat atas perbedaan konfigurasi antar environment.
- Bisa diintegrasikan ke pipeline CI/CD sebagai gate otomatis.
- Jadi portofolio open source yang menunjukkan kemampuan Go + pemahaman DevOps.

## 3. Target Pengguna

- Developer individu/tim kecil yang self-host di VPS tanpa secret manager enterprise (Vault, dsb).
- Maintainer proyek open source yang ingin kontributor gampang setup `.env` lokal.
- Tim DevOps kecil yang butuh validasi env sebagai bagian dari CI/CD.

## 4. Lingkup Fitur

### 4.1 MVP (Fase 1 — wajib ada untuk rilis pertama)

| # | Fitur | Deskripsi |
|---|-------|-----------|
| 1 | **Diff checker** | Bandingkan `.env.example` vs `.env` lokal: variable hilang, extra, atau kosong. |
| 2 | **Multi-environment compare** | Bandingkan `.env.local` vs `.env.staging` vs `.env.production` sekaligus, output dalam bentuk tabel. |
| 3 | **Remote sync check (SSH)** | Ambil `.env` dari VPS via SSH, bandingkan dengan versi lokal/repo. |
| 4 | **CI/CD exit code gate** | Exit code non-zero jika ada mismatch kritis — bisa dipasang di GitHub Actions sebelum step deploy. |

**Kriteria selesai Fase 1:** `envman check` bisa dijalankan di repo apa pun, mendeteksi minimal 3 jenis masalah (missing, extra, empty), dan bisa dipakai sebagai step CI.

### 4.2 Fase 2 — Validasi & Keamanan

| # | Fitur | Deskripsi |
|---|-------|-----------|
| 5 | **Type/format validation** | Validasi value sesuai format yang diharapkan (URL, port number, boolean) berdasarkan naming convention atau schema opsional (`.envman.yaml`). |
| 6 | **Secret leak scanner** | Deteksi value yang terlihat seperti secret asli (pola API key/token) di file `.env.example`. |
| 7 | **Required vs optional flagging** | Tandai variable wajib vs opsional berdasarkan komentar di `.env.example` (mis. `# required`). |

### 4.3 Fase 3 — Integrasi & Automasi

| # | Fitur | Deskripsi |
|---|-------|-----------|
| 8 | **Docker/Compose aware** | Baca juga `environment:` di `docker-compose.yml`, bukan cuma file `.env`. |
| 9 | **Sync command** | `envman push` / `envman pull` untuk menyinkronkan variable yang berbeda ke VPS (dengan konfirmasi eksplisit sebelum overwrite). |
| 10 | **Report output** | Export hasil ke Markdown/JSON, opsional kirim notifikasi ke Telegram. |

### 4.4 Nice-to-have (belum prioritas)
- Dukungan baca dari secret manager lain (Doppler, 1Password CLI).
- Plugin/extension system untuk validator custom.

## 5. Alur Penggunaan Utama (User Flow)

```
$ envman check
→ Membaca .env.example di direktori saat ini
→ Membandingkan dengan .env lokal
→ Menampilkan tabel: [MISSING] [EXTRA] [EMPTY] [OK]
→ Exit code 0 jika semua OK, 1 jika ada masalah kritis

$ envman check --remote user@vps:/path/to/app
→ SSH ke VPS, ambil .env di sana
→ Bandingkan dengan .env lokal/repo
→ Tampilkan diff

$ envman check --ci
→ Mode ringkas untuk CI/CD, output minim, exit code jadi sinyal utama
```

## 6. Kebutuhan Teknis (Non-Fungsional)

- **Bahasa:** Go (single binary, mudah didistribusikan, cocok untuk CLI DevOps).
- **Distribusi:** Binary release via GitHub Releases + install script (`curl | sh`), opsional Homebrew tap.
- **Dependensi SSH:** gunakan library Go native (`golang.org/x/crypto/ssh`), hindari shell-out ke `ssh` binary agar portable.
- **Konfigurasi:** file opsional `.envman.yaml` untuk aturan validasi custom.
- **Kompatibilitas:** Linux & macOS minimal (VPS budget biasanya Linux); Windows nice-to-have.
- **Performa:** Pengecekan lokal harus instan (<1 detik untuk file wajar); remote check dibatasi timeout SSH yang jelas.

## 7. Metrik Keberhasilan

- Tool berhasil dipakai sendiri (dogfooding) di minimal 1 proyek nyata (mis. school-website atau GradeSnap) tanpa masalah.
- Terintegrasi sukses di 1 pipeline GitHub Actions sebagai gate.
- Repo mendapat engagement dasar (star, issue, atau kontribusi) sebagai indikator utilitas ke komunitas.

## 8. Di Luar Cakupan (Out of Scope)

- Manajemen secret penuh (bukan pengganti Vault/Doppler) — fokus hanya pada *checking & syncing*, bukan penyimpanan terenkripsi.
- UI web/dashboard — ini murni CLI.
- Auto-fix otomatis tanpa konfirmasi (semua aksi tulis/overwrite harus eksplisit dikonfirmasi user).

## 9. Roadmap Ringkas

| Fase | Fokus | Estimasi |
|------|-------|----------|
| 1 | MVP: diff checker, multi-env compare, remote check, CI gate | Rilis awal |
| 2 | Validasi format, secret scanner, required flagging | Setelah MVP stabil dipakai sendiri |
| 3 | Docker-compose aware, sync command, report/Telegram | Setelah ada beberapa pengguna eksternal |

## 10. Risiko & Mitigasi

| Risiko | Mitigasi |
|--------|----------|
| SSH ke VPS production berisiko kalau ada bug di fitur sync/push | Default read-only; fitur push butuh flag eksplisit + konfirmasi |
| Overlap dengan tool sejenis (dotenv-linter, dll) | Diferensiasi lewat fitur remote-check via SSH dan integrasi Telegram/CI yang lebih spesifik ke workflow self-hosting |
| Scope creep ke secret manager penuh | Tegas jaga di "checker & syncer", bukan "storage" |
