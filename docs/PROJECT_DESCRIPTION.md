# 📺 Video Storage AI – Project Specification

## 💻 System Environment

### 🧩 Hardware

-   **CPU:** Intel Core i7-14700K
-   **Memory:** 32 GB DDR5
-   **Storage:** Samsung 990 PRO 2 TB NVMe M.2
-   **Network:** 1 Gbit Ethernet
-   **GPU:** NVIDIA RTX 4080 Super (latest Game Ready Drivers)

### ⚙️ Software

-   **FFmpeg:** 8.0 (full build from [www.gyan.dev](http://www.gyan.dev))
-   **VLC Media Player:** 3.0.21 “Vetinari”
-   **K-Lite Codec Pack:** 19.1.5 Full
-   **Frontend Stack:** Vue 3 (Composition API), Bootstrap 5.3, Font Awesome (referenced in `package.json`)

---

## 📝 Overview

**Video Storage AI** is a **Windows desktop application** built with **Vue 3** and **Go**, designed to function as a **video browser**, **player**, and **AI-powered media organizer**.

It provides a **YouTube-like browsing experience** for local video collections while offering **intelligent organization tools**, **metadata management**, **tag management**, **performer and studio presentation**, and a **Live App Activity Monitor** — all running **offline**.

---

## 🎬 Background & Motivation

The user currently maintains a **Jellyfin server** with multiple libraries:

-   🎥 **Movies & TV Shows**
-   ✂️ **Edited Videos** (finished projects)
-   🗂️ **Backup Videos** (raw/original sources)

### 🧠 The Challenge

With over **500 videos** across multiple folders and frequent new additions, traditional tools like **Windows File Explorer** are **slow, inefficient, and not scalable**.
There is a clear need for a **dedicated, performance-optimized solution** that can intelligently organize and present large video libraries.

---

## 🎯 Goals

-   Replace traditional file browsing with a **video-centric UI**
-   Integrate **metadata display**, **thumbnail previews**, and **in-app playback**
-   Provide **AI-powered assistance** for:

    -   File and folder naming conventions
    -   Folder structure optimization
    -   Large-scale content management
    -   Library planning (tagging, categorization, restructuring)

-   Support **direct file operations** (rename, move, delete, reorganize)
-   Deliver a **YouTube-like local browsing experience**
-   Implement a **high-speed caching and indexing system**
-   Introduce **performer presentation system**
-   Introduce **studio and group management**
-   Include **tag management for videos**
-   Include a **Live App Activity Monitor** for real-time insight into:

    -   Background processes (scanning, AI analysis, indexing, caching)
    -   Active tasks and system resource usage
    -   Performance metrics and current app operations

---

## ⚙️ Core Features

### 📂 Video Browser Interface

-   Clean, **dark-themed**, YouTube-inspired design
-   Hierarchical folder navigation (recursive subfolder support)
-   Advanced **search and filtering** options
-   Dynamic thumbnail previews

### ▶️ Integrated Video Player

-   Built-in playback using system codecs or FFmpeg
-   Displays technical metadata (codec, resolution, duration, bitrate, etc.)
-   Fast video scrubbing with preview thumbnails
-   Resume playback and bookmarking support

### 🧠 AI Assistant

-   Acts as a **personal organizing companion**
-   Suggests **naming conventions**, **folder structures**, and **metadata corrections**
-   Identifies **duplicates**, **unfinished edits**, or **misplaced files**
-   Provides **optimization insights** for better scalability
-   Offers **auto-tagging** suggestions for videos during import or metadata refresh
-   Integrates with **AdultDataLink API** for fetching performer and scene metadata (configuration in `adultdatalinkapi.json`)
-   Engages in **live chat-style communication** with the user

### 🧍 Performer Presentation & Management

-   Dedicated **Performer Page** with a **Performer Wall** displaying:

    -   Each performer as a card with a looping `.mkv` video preview
    -   Performer’s name below the preview and **scene count badge**

-   Clicking a performer opens a **Performer Details Panel** from the side:

    -   **Carousel** at the top with large focused `.mkv` preview and smaller thumbnails below
    -   Detailed metadata (name, tags, appearances, folder path, last updated date)

-   **Right-click context menu**:

    -   `Fetch Metadata` → pulls performer and scene metadata from **AdultDataLink API**
    -   `Reset Metadata`
    -   `Reset Previews`
    -   `Delete Performer` (with confirmation dialog)

-   Supports **real-time updates** when metadata or previews change

### 🏷️ Tag Management

-   Centralized **Tag Management Dashboard** for creating, editing, and deleting tags
-   Tags apply **only to videos**
-   Features:

    -   Custom color labels and icons (Font Awesome)
    -   Tag merging and renaming
    -   Multi-select tagging support in batch operations

-   Search and filter by tags directly in the video browser
-   Planned: **AI auto-tagging** suggestions during video import or metadata refresh

### 🏢 Studio & Group Management

#### 🎬 Studio Management

-   Dedicated **Studio Page** with a **Studio Wall**:

    -   Each studio card displays logo, name, and total videos count
    -   Optional short description or tagline

-   **Studio Details Panel** includes:

    -   Overview (name, logo, description, founded date, country)
    -   Associated performers and videos
    -   Groups (child entities) displayed in collapsible list
    -   Quick actions: `Fetch Metadata`, `Reset Metadata`, `Delete Studio`

-   Right-click context menu similar to performers

#### 🧩 Group Management

-   Groups belong to **Studios** and represent sub-labels or content series
-   Each group card shows name, optional logo, and total video count
-   **Group Details Panel** mirrors the studio/performer pattern:

    -   List of videos and performers in the group
    -   Metadata fields for internal organization

-   Supports **hierarchical filtering**: Studio → Group → Videos

### 🛠️ Organization Tools

-   Rename, move, and delete videos directly within the app
-   Batch operations for new files
-   **Drag-and-drop** reorganization
-   **AI-guided sorting** for large imports

### 📊 Live App Activity Monitor

-   Real-time dashboard for internal operations: scanning, indexing, AI tagging, caching, metadata updates
-   Task display: progress bar, time estimate, and live status icon (Font Awesome)
-   Performance charts: CPU/GPU usage, disk I/O, thread activity, library scanning throughput
-   Filtered log view for debugging with tabs for AI Assistant, File Scanner, Metadata Engine
-   Built using **Bootstrap cards** and **Vue reactive dashboard** for smooth live updates

### 🚀 Performance & Scalability

-   Optimized for **500+ videos**, scalable to thousands
-   Efficient **multi-threaded scanning** and **metadata caching**
-   Incremental updates to avoid redundant rescanning

## 🎨 UI / UX Concept

### 🧰 Frontend Tech Stack

-   **Vue 3 (Composition API)**
-   **Bootstrap 5.3** for responsive grid and layout
-   **Font Awesome** for icons _(defined in `package.json`)_
-   Custom **dark theme** with smooth transitions via `<Transition>` and `<TransitionGroup>`

### 🧍 Performer Interface

-   **Performer Wall**: grid of performer cards with `.mkv` previews, names, scene count badges
-   **Details Panel**: slides from the right, includes carousel of video previews and metadata, context menu with API actions
-   Dynamic updates and smooth animations for hover, selection, and metadata refresh

### 🏢 Studios & Groups Interface

-   **Studio Wall**: grid layout with logo, name, total content count
-   **Studio Details Panel**: overview, performers, groups, videos; collapsible groups
-   **Group Cards & Panel**: nested within studios, show video and performer list, metadata fields
-   Hierarchical filtering: Studio → Group → Videos

### 🏷️ Tag Management UI

-   Bootstrap tables and tag chips for easy management
-   Color-coded badges, inline editing, drag-and-drop ordering
-   Quick search integration (`#tagname` filters videos instantly)

### 📊 Live App Activity Monitor

-   Sidebar or modal window with progress bars, charts, and logs
-   Interactive filtering and real-time updates for active operations, AI tasks, scanning, and metadata extraction
