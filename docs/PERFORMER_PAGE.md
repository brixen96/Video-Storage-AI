# 🎭 Performer Page Specification

## Overview

The Performer Page provides a visual and interactive way to browse, manage, and view metadata for all performers in the system.
It automatically synchronizes with the **local `assets/` folder** and aligns metadata with the **AdultDataLink API**, ensuring seamless integration between local assets and fetched data.

---

## 🧩 UI Components

### **1. Performer Wall (Grid View)**

-   **Layout:** CSS Grid using `auto-fill` with responsive column widths.
-   **Card Contents:**

    -   Looping `.mkv` video preview (from `primary_preview`).
    -   Performer name centered below the preview.
    -   Scene count badge in **top-right** corner.
    -   16:9 aspect ratio (no crop; scale to fit).
    -   Hover effects:

        -   Subtle elevation (`box-shadow`)
        -   Glow effect around the card border

-   **Interaction:**

    -   Left-click → opens **Performer Details Panel**
    -   Right-click → opens **Context Menu**

---

### **2. Performer Details Panel (Side Drawer)**

-   **Behavior:**

    -   Slides in from the **right side**.
    -   Click outside overlay to close.

-   **Contents:**

    -   **Header:** Performer name + “zoo” toggle (radio button).
    -   **Top Section (Carousel):**

        -   Large primary preview video.
        -   Smaller previews (thumbnails or looping `.mkv`) below.
        -   Clicking another preview updates the main player.

    -   **Metadata Grid:**

        -   Basic info: name, age, alias, nationality, etc.
        -   Appearance details (from `appearance` object).
        -   Performance tags (from `performances`).

    -   **Action Buttons:**

        -   `Fetch Metadata` → syncs from AdultDataLink API
        -   `Reset Metadata` → clears performer metadata
        -   `Reset Previews` → refreshes local previews from asset folder
        -   `Delete Performer` → removes entry from DB and local index

            -   includes delete confirmation modal

---

### **3. Context Menu (Right-Click)**

-   Appears at cursor position.
-   Menu items:

    -   **Fetch Metadata**
    -   **Reset Metadata**
    -   **Reset Previews**
    -   **Delete Performer** (with confirmation)

-   Should dismiss on click outside or ESC key.

### **4. Search, Sort & Filter, Show Zoo Only**

-   Real-time client-side search by **name**.
-   Sort by: Name, Age, Breast Size, Height, Scene Count. Sort applies auto always.
-   Filters: Show Zoo Performers Only (true or false), Min/max Age, Min/max Breast Size, min/max Height
-   Apply filters button.
-   Clear button to reset results.

## 🔗 API Integration

### **Backend Endpoints**

| Operation                         | Method | Description                              |
| --------------------------------- | ------ | ---------------------------------------- |
| `GET /api/performers`             | GET    | Get all performers                       |
| `POST /api/performers`            | POST   | Add a new performer                      |
| `PUT /api/performers/{id}`        | PUT    | Update performer metadata                |
| `DELETE /api/performers/{id}`     | DELETE | Remove performer                         |
| `POST /api/performers/{id}/fetch` | POST   | Fetch metadata from AdultDataLink        |
| `POST /api/performers/scan`       | POST   | Scan `assets/` folder for new performers |

### **Error Handling**

-   Graceful fallback if AdultDataLink API fails.
-   Show toast notifications for all success/error states.
-   Retry or refresh options for network/API errors.

## 🧐 Metadata Sync (AdultDataLink)

-   **API Endpoint:**
    `https://api.adultdatalink.com/pornstar/pornstar-data?name={PerformerName}`
    -   NOTE: If the Full name has a first name and a lastname the space should replaced by %20 (https://api.adultdatalink.com/pornstar/pornstar-data?name=Aidra20%Fox)
-   **Header:**
    `Authorization: ${ADULTDATALINK_API_KEY}` (from `.env`)
-   Response JSON is stored **as-is** in the performer’s DB record.
-   **Merging Strategy:**
    -   Existing local properties are updated, not overwritten unless blank.
    -   New data appended to arrays (e.g., external links, tags).

---

## 📦 Performer Object Structure

### **Final Schema Sample of Performer "Aidra Fox"**

```json
{
	"appearance": {},
	"performances": {},
	"social_media": {},
	"platform_views": {},
	"platform_video_counts": {},
	"platform_profile_counts": {},
	"tags": [],
	"external_links": [],
	"bios": {},
	"country": "",
	"avatar": "",
	"subscribers": null,
	"rating": null,
	"official_website": "",
	"name": "Aidra Fox",
	"image_url": "",
	"rank": "",
	"age": "",
	"aliases": "",
	"date_of_birth": "",
	"place_of_birth": "",
	"career_start": "",
	"nationality": "",
	"total_views": null,
	"total_video_count": null,
	"total_platform_hits": null,
	"previews": [],
	"primary_preview": "",
	"zoo": false
}
```

---

## ⚙️ Initialization Workflow

### **1. Startup Scan**

When the app starts:

-   Scan `/api/assets/` for folders.
-   Each folder name = Performer name.
-   Check if the performer exists in DB:

    -   **No:** create a new record using the above schema.
    -   **Yes:** validate previews and metadata consistency.

### **2. Previews**

-   `.mkv` files inside the performer’s folder are auto-loaded into the `previews` array.
-   The first file becomes `primary_preview` by default (can be changed later in UI).

### **3. Metadata Fetch**

-   Triggered manually via button or context menu.
-   Fetched data merged into local record.
