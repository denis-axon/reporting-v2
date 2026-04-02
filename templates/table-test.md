# Table Rendering Test

## 1. Basic Table (Default Alignment)

| ID | Name       | Role       |
|----|------------|------------|
| 1  | Alice Smith | Engineer  |
| 2  | Bob Jones   | Designer  |
| 3  | Carol White | Manager   |

---

## 2. Column Alignment Test

| Left Aligned | Center Aligned | Right Aligned |
|:-------------|:--------------:|--------------:|
| apple        |     banana     |         cherry |
| dog          |      cat       |          bird |
| 100          |      200       |           300 |

---

## 3. Inline Formatting Inside Cells

| Feature       | Status        | Notes                         |
|---------------|---------------|-------------------------------|
| **Bold text** | ✅ Supported   | Works in all cells            |
| *Italic text* | ✅ Supported   | Use `*` or `_`                |
| `inline code` | ⚠️ Check PDF  | May vary by renderer          |
| ~~Strikethrough~~ | ❓ Test   | Not all renderers support it  |

---

## 4. Numeric & Date Data

| Order ID | Product        | Quantity | Unit Price | Total    | Order Date  |
|----------|----------------|----------|------------|----------|-------------|
| 1001     | Laptop Pro     | 2        | $1,299.99  | $2,599.98 | 2026-01-15 |
| 1002     | Wireless Mouse | 5        | $29.99     | $149.95  | 2026-02-03  |
| 1003     | USB-C Hub      | 3        | $49.99     | $149.97  | 2026-03-22  |
| 1004     | Mechanical Keyboard | 1   | $189.00    | $189.00  | 2026-04-01  |

---

## 5. Long Text / Wrapping Test

| Parameter       | Description                                                                 |
|-----------------|-----------------------------------------------------------------------------|
| `timeout`       | The maximum number of milliseconds to wait before the request is aborted.   |
| `retryCount`    | Number of times the client will retry a failed request before giving up.    |
| `baseURL`       | The base URL prepended to all relative paths when making HTTP requests.     |
| `authToken`     | Bearer token sent in the Authorization header for authenticated endpoints.  |

---

## 6. Mixed Data Types

| Username   | Age | Score  | Active | Joined      |
|------------|-----|--------|--------|-------------|
| john_doe   | 28  | 98.50  | Yes    | 2023-05-10  |
| jane_smith | 34  | 74.25  | No     | 2021-11-22  |
| dev_bob    | 22  | 100.00 | Yes    | 2025-08-01  |
| test_user  | 45  | 55.75  | Yes    | 2020-03-15  |

---

## 7. Single Column Table

| Countries         |
|-------------------|
| United States     |
| Germany           |
| Japan             |
| Brazil            |

---

## 8. Wide Table (Many Columns)

| A | B | C | D | E | F | G | H |
|---|---|---|---|---|---|---|---|
| 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
| a | b | c | d | e | f | g | h |
| ✓ | ✗ | ✓ | ✗ | ✓ | ✗ | ✓ | ✗ |