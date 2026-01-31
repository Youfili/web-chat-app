# 💬 WASAtext – Real-Time Messaging Application

This repository contains the implementation of a **web-based chat application** developed as part of the *Web and Software Architecture* course project at [Sapienza University of Rome](http://gamificationlab.uniroma1.it/en/wasa/).

**WASAtext** allows users to communicate in real time through an intuitive and responsive web interface.  
The project focuses on applying key concepts of web architecture, including client-server communication, RESTful APIs, and real-time data exchange.

> This project was developed for educational purposes and is **not intended for production environments**.

## 🚀 Key Features

### 📱 Core Messaging
* **Real-time Communication:** Instant messaging with auto-refresh (polling) strategy.
* **Private & Group Chats:** Seamlessly switch between 1-on-1 conversations and multi-user groups.
* **Rich Media Support:** Send text, emojis, and **images**.
* **Interactive Features:** Reply to specific messages, forward messages to other chats, and react with emojis.

### ✅ Advanced Read Receipts (The "Checkmarks" Logic)
Implemented a robust state tracking system similar to WhatsApp:
* **✓ (Sent):** Message saved on the server.
* **✓✓ (Delivered - Gray):** Message received by the recipient's client (or all group members).
* **✓✓ (Read - Blue):**
    * *Private:* The recipient has opened the chat.
    * *Groups:* **All** participants have opened the chat and seen the message.

### 👥 Group Management
* **Dynamic Group Names:** If a group has no name, it automatically displays a list of members (e.g., "Alice, Bob, Charlie").
* **Admin System:** Creators are admins by default. Admins can promote/demote others.
* **Membership Control:** Add/Remove users. Users can leave groups (with safety checks so the last admin cannot leave without promoting someone else).

### 🛠 Technical Highlights
* **Robust Backend:** Written in Go, featuring atomic database transactions to ensure data consistency.
* **Smart Polling:** Optimized frontend state management to handle user removal from groups or chat deletions in real-time.
* **Input Validation:** Sanitzed usernames (trimmed spaces) and safe SQL queries to prevent injection.
* **Dockerized:** Ready-to-run environment with zero configuration.

---

## 🛠 Tech Stack

* **Backend:** Go (Golang) 1.19+
* **Frontend:** Vue.js 3 (Options API) + Bootstrap 5
* **Database:** SQLite3
* **Containerization:** Docker & Docker Compose
* **API Protocol:** REST (JSON)

---

## 📦 How to Run (Recommended)

The easiest way to run the application is using **Docker Compose**. This will set up both the backend (API) and the frontend (Web Server) automatically.

### Prerequisites
* [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed.

### Steps
1.  Clone the repository:
    ```bash
    git clone [https://github.com/YOUR_USERNAME/WASAtext.git](https://github.com/YOUR_USERNAME/WASAtext.git)
    cd WASAtext
    ```

2.  Start the application:
    ```bash
    docker-compose up --build
    ```

3.  Access the app:
    Open your browser and navigate to: **http://localhost:8080**

---

## 🧪 Development & Manual Build

If you want to run it without Docker for development purposes:

### Backend
```bash
go run ./cmd/webapi/
# API listens on localhost:3000
```

### Frontend
```bash
cd webui
npm install (or yarn install)
npm run dev (or yarn run dev)
```

## Project structure

* `cmd/` contains all executables; Go programs here should only do "executable-stuff", like reading options from the CLI/env, etc.
	* `cmd/healthcheck` is an example of a daemon for checking the health of servers daemons; useful when the hypervisor is not providing HTTP readiness/liveness probes (e.g., Docker engine)
	* `cmd/webapi` contains an example of a web API server daemon
* `demo/` contains a demo config file
* `doc/` contains the documentation (usually, for APIs, this means an OpenAPI file)
* `service/` has all packages for implementing project-specific functionalities
	* `service/api` contains an example of an API server
	* `service/globaltime` contains a wrapper package for `time.Time` (useful in unit testing)
* `vendor/` is managed by Go, and contains a copy of all dependencies
* `webui/` is an example of a web frontend in Vue.js; it includes:
	* Bootstrap JavaScript framework
	* a customized version of "Bootstrap dashboard" template
	* feather icons as SVG
	* Go code for release embedding

Other project files include:
* `open-node.sh` starts a new (temporary) container using `node:20` image for safe and secure web frontend development (you don't want to use `node` in your system, do you?).

## Go vendoring

This project uses [Go Vendoring](https://go.dev/ref/mod#vendoring). You must use `go mod vendor` after changing some dependency (`go get` or `go mod tidy`) and add all files under `vendor/` directory in your commit.

For more information about vendoring:

* https://go.dev/ref/mod#vendoring
* https://www.ardanlabs.com/blog/2020/04/modules-06-vendoring.html

## Node/YARN vendoring

This repository uses `yarn` and a vendoring technique that exploits the ["Offline mirror"](https://yarnpkg.com/features/caching). As for the Go vendoring, the dependencies are inside the repository.

You should commit the files inside the `.yarn` directory.

## How to build

If you're not using the WebUI, or if you don't want to embed the WebUI into the final executable, then:

```shell
go build ./cmd/webapi/
```

If you're using the WebUI and you want to embed it into the final executable:

```shell
./open-node.sh
# (here you're inside the container)
yarn run build-embed
exit
# (outside the container)
go build -tags webui ./cmd/webapi/
```

## How to run (in development mode)

You can launch the backend only using:

```shell
go run ./cmd/webapi/
```

If you want to launch the WebUI, open a new tab and launch:

```shell
./open-node.sh
# (here you're inside the container)
yarn run dev
```

## How to build for production / homework delivery

```shell
./open-node.sh
# (here you're inside the container)
yarn run build-prod
```

For "Web and Software Architecture" students: before committing and pushing your work for grading, please read the section below named "My build works when I use `yarn run dev`, however there is a Javascript crash in production/grading"

## Known issues

### My build works when I use `yarn run dev`, however there is a Javascript crash in production/grading

Some errors in the code are somehow not shown in `vite` development mode. To preview the code that will be used in production/grading settings, use the following commands:

```shell
./open-node.sh
# (here you're inside the container)
yarn run build-prod
yarn run preview
```

## License

See [LICENSE](LICENSE).
