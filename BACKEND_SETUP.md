# Backend Setup Guide

To enable **Secure Upload** (S3 upload + view-only link), run the backend API server. The bucket name and optional secrets live in a **`.env` file** (set once, not committed). AWS credentials are used from your **terminal login** (AWS CLI).

---

## 1. Set the bucket name once (`.env`)

The backend reads `AWS_BUCKET_NAME` from a `.env` file in the `backend` folder so you never have to type it in the terminal.

1. Go to the backend directory:
   ```bash
   cd backend
   ```
2. Create your env file from the example:
   ```bash
   cp .env.example .env
   ```
3. Edit `.env` and set your real bucket name:
   ```bash
   AWS_BUCKET_NAME=your-actual-bucket-name
   ```
   (Replace `your-actual-bucket-name` with the S3 bucket you create in the next step.)

`.env` is in `.gitignore` and is not committed. You only set this once per machine.

---

## 2. Set up AWS login on your terminal

The backend uses the same credentials as the AWS CLI. Set them up once on your machine.

### 2a. Install AWS CLI

- **macOS (Homebrew):**
  ```bash
  brew install awscli
  ```
- **Windows:** [Install AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html).
- **Linux:**
  ```bash
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
  unzip awscliv2.zip && sudo ./aws/install
  ```

### 2b. Log in (configure credentials)

Run:

```bash
aws configure
```

You’ll be prompted for:

| Prompt | What to enter |
|--------|----------------|
| **AWS Access Key ID** | From AWS Console → IAM → Your user → Security credentials → Create access key |
| **AWS Secret Access Key** | From the same “Create access key” step |
| **Default region** | Your bucket’s region, e.g. `us-east-1` |
| **Default output format** | Press Enter (or use `json`) |

Credentials are stored in `~/.aws/credentials`. The backend (and any tool using the AWS SDK) will use them automatically; no need to put them in `.env` unless you want to override.

### 2c. Create an S3 bucket (if you don’t have one)

1. In [AWS Console](https://console.aws.amazon.com/) go to **S3** → **Create bucket**.
2. Pick a **bucket name** and **region** (e.g. `us-east-1`).
3. Create the bucket, then put that **exact bucket name** in `backend/.env` as `AWS_BUCKET_NAME` (step 1).

---

## 3. Initialize and run the backend

From the **project root** (or from `backend`):

```bash
cd backend
go mod init backend   # only first time
go mod tidy
go run main.go
```

You do **not** need to `export AWS_BUCKET_NAME` in the terminal; it’s read from `backend/.env`. The server will use your AWS CLI credentials and listen on `http://localhost:8080`.

---

## 4. How it works

1. **Upload**: User selects a file (Secure Upload on a page or in the extension popup). The extension asks your backend for a presigned URL, then uploads the file directly to S3.
2. **View-only link**: The backend returns a one-time link (e.g. `http://localhost:8080/view/<id>`). Opening it once shows the file in the browser (inline when possible); after that the link expires.

---

## 5. Test

1. Reload the extension at `chrome://extensions`.
2. Use **Secure Share** on a page with a file input, or the extension popup → **Secure Upload**.
3. After upload, copy or open the view-only link and confirm it opens once.

---

## Optional: override in `.env`

You can put more in `backend/.env` if you want (still do not commit it):

- `VIEW_LINK_BASE_URL` – base URL for view links (default `http://localhost:8080`).
- `AWS_REGION` – if you don’t want to rely on `aws configure` default.
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` – only if you don’t want to use `~/.aws/credentials`.
