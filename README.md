
# November Project Overview

This repository contains the work of team **November** for the [Black Forest Hackathon 2025](https://www.blackforesthackathon.de/challenges-sxh25/).

The project explores digital health innovation using synthetic FHIR clinical data and a modern mobile-first frontend. It consists of:

- **backend/**: A Healthcare MCP (Model Context Protocol) server implementation in Go that provides healthcare-related tools. It integrates with FHIR healthcare data stored in SQLite and uses OpenRouter API for AI-powered responses. Features include patient lookup, appointment scheduling, medical history retrieval, medication information, and health question answering.
- **smartphone-app/**: A Vue.js-based PWA demonstrating a chat-style interface for patient interaction, including text and simulated audio messages.
- **data/**: Additional datasets or resources (if present).

See each subproject's README for technical details and setup.

To inspect the demo data, download [this example database](https://drive.google.com/file/d/1Qe5xsy0VTOGGOLZa8SykeiHGATrisWqi/view?usp=sharing) and open it locally with [a sqlite viewer](https://inloop.github.io/sqlite-viewer/).
