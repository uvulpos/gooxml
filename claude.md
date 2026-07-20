# ODT Table of Contents — Implementierungsreferenz

Referenz zum korrekten Einfügen eines Inhaltsverzeichnisses (Table of Contents / TOC)
in ODT-Dokumenten (OpenDocument Format). Grundlage: OASIS OpenDocument Specification
(ODF 1.2 / 1.3 / 1.4). Gedacht zur Weiterverarbeitung in einem Pull Request.

---

## 1. Kernprinzip

Ein Inhaltsverzeichnis in ODT ist **kein einzelner Block**, sondern das Element
`<text:table-of-content>` mit **zwei Pflichtteilen in fester Reihenfolge**:

1. `<text:table-of-content-source>` — die **Vorlage** (definiert, *wie* das TOC regeneriert wird).
2. `<text:index-body>` — der **vorab generierte, sichtbare Inhalt** (wird angezeigt, bis der Nutzer "Verzeichnis aktualisieren" wählt).

> Die falsche/fehlende zweite Hälfte oder eine falsche Reihenfolge ist die häufigste Fehlerquelle.

---

## 2. Platzierung im Dokument (Schema-Ebene)

- `<text:table-of-content>` ist ein Element auf **body-text / paragraph level**.
- Es MUSS direkt unter `<office:text>` oder innerhalb einer `<text:section>` stehen.
- Es DARF **NICHT** in ein `<text:p>` verschachtelt werden → sonst Schema-Validierungsfehler.

---

## 3. Vollständige Zielstruktur (Minimalbeispiel)

```xml
<text:table-of-content text:style-name="Sect1"
                       text:protected="true"
                       text:name="Table of Contents1">

  <!-- TEIL 1: Vorlage zur Neugenerierung -->
  <text:table-of-content-source text:outline-level="10"
                                text:use-outline-level="true">
    <text:index-title-template text:style-name="Contents_20_Heading">Inhaltsverzeichnis</text:index-title-template>

    <!-- Ein Entry-Template PRO Gliederungsebene (1..outline-level) -->
    <text:table-of-content-entry-template text:outline-level="1"
                                          text:style-name="Contents_20_1">
      <text:index-entry-link-start text:style-name="Internet_20_Link"/>
      <text:index-entry-chapter/>
      <text:index-entry-text/>
      <text:index-entry-tab-stop style:type="right" style:leader-char="."/>
      <text:index-entry-page-number/>
      <text:index-entry-link-end/>
    </text:table-of-content-entry-template>

    <text:table-of-content-entry-template text:outline-level="2"
                                          text:style-name="Contents_20_2">
      <text:index-entry-link-start text:style-name="Internet_20_Link"/>
      <text:index-entry-chapter/>
      <text:index-entry-text/>
      <text:index-entry-tab-stop style:type="right" style:leader-char="."/>
      <text:index-entry-page-number/>
      <text:index-entry-link-end/>
    </text:table-of-content-entry-template>
    <!-- Ebenen 3..10 analog -->
  </text:table-of-content-source>

  <!-- TEIL 2: Vorab generierter, sichtbarer Inhalt -->
  <text:index-body>
    <text:index-title text:style-name="Sect1"
                      text:name="Table of Contents1_Head">
      <text:p text:style-name="Contents_20_Heading">Inhaltsverzeichnis</text:p>
    </text:index-title>

    <text:p text:style-name="Contents_20_1">
      <text:a xlink:type="simple" xlink:href="#__RefHeading__1"
              text:style-name="Internet_20_Link">1. Einleitung<text:tab/>1</text:a>
    </text:p>
    <text:p text:style-name="Contents_20_2">
      <text:a xlink:type="simple" xlink:href="#__RefHeading__2"
              text:style-name="Internet_20_Link">1.1 Motivation<text:tab/>2</text:a>
    </text:p>
  </text:index-body>

</text:table-of-content>
```

---

## 4. Wichtige Elemente & Attribute

| Element / Attribut | Bedeutung |
|---|---|
| `text:table-of-content` | Wurzelelement des TOC. |
| `text:name` | Eindeutiger Name des Verzeichnisses (z. B. `Table of Contents1`). |
| `text:protected="true"` | Schützt vor versehentlichem manuellem Editieren. |
| `text:table-of-content-source` | Vorlage/Definition des Verzeichnisses (Teil 1). |
| `text:outline-level="10"` | Bis zu welcher Überschriftenebene erfasst wird (max. 10). |
| `text:use-outline-level="true"` | TOC wird aus den Gliederungsebenen der Überschriften erzeugt. |
| `text:index-title-template` | Format/Titel-Vorlage des Verzeichnisses. |
| `text:table-of-content-entry-template` | **Ein Template pro Ebene**; `text:outline-level` gibt die Ebene an. |
| `text:index-entry-chapter` | Fügt Kapitelnummer ein. |
| `text:index-entry-text` | Fügt den Überschriftentext ein. |
| `text:index-entry-tab-stop` | Tab (z. B. rechtsbündig mit Füllzeichen `.`) vor der Seitenzahl. |
| `text:index-entry-page-number` | Fügt die Seitenzahl ein. |
| `text:index-entry-link-start` / `-link-end` | Umschließt einen Eintrag als klickbaren Link. |
| `text:index-body` | Sichtbarer, vorab generierter Inhalt (Teil 2). |
| `text:index-title` | Titelblock innerhalb von `index-body`. |

---

## 5. Style-Namen (Namenskodierung beachten)

- Style-Referenzen wie `Contents_20_1` sind ODF-kodiert: `_20_` = Leerzeichen.
  → `Contents_20_1` entspricht dem Stil **"Contents 1"**, `Contents_20_Heading` = "Contents Heading".
- Alle referenzierten Absatz-/Zeichenstile (`Contents 1`..`Contents N`, `Contents Heading`,
  `Internet Link`) müssen in `styles.xml` oder `content.xml` **definiert** sein,
  sonst je nach Validator/Reader Fehler oder fehlende Formatierung.

---

## 6. Verlinkung & Seitenzahlen (Anker erforderlich)

Damit Links und Seitenzahlen funktionieren, müssen die Überschriften **Anker** besitzen,
auf die die `xlink:href` der TOC-Einträge zeigen:

```xml
<text:h text:style-name="Heading_20_1" text:outline-level="1">
  <text:bookmark-start text:name="__RefHeading__1"/>Einleitung<text:bookmark-end text:name="__RefHeading__1"/>
</text:h>
```

- `xlink:href="#__RefHeading__1"` im TOC-Eintrag muss exakt zum Bookmark-Namen passen.
- Seitenzahlen im `index-body` sind statisch; sie werden erst beim "Aktualisieren"
  im Office-Programm neu berechnet.

---

## 7. Checkliste für den Generator / PR

- [ ] `<text:table-of-content>` steht direkt unter `<office:text>` oder in `<text:section>`, **nicht** in `<text:p>`.
- [ ] Reihenfolge korrekt: erst `<text:table-of-content-source>`, dann `<text:index-body>`.
- [ ] `text:outline-level` gesetzt und `text:use-outline-level="true"`.
- [ ] Für **jede** erfasste Ebene ein `text:table-of-content-entry-template` (Ebenen 1..N lückenlos).
- [ ] `<text:index-body>` erzeugt (sonst leeres TOC bis zum manuellen Update).
- [ ] Alle referenzierten Styles existieren in `styles.xml`/`content.xml`.
- [ ] Jede Überschrift hat einen Bookmark-Anker; jeder TOC-Eintrag verweist per `xlink:href` darauf.
- [ ] `text:name` des TOC ist eindeutig im Dokument.
- [ ] Namespaces `text:`, `style:`, `xlink:` sind im Wurzelelement deklariert.
- [ ] Ausgabe gegen das ODF-RNG-Schema validiert.

---

## 8. Typische Fehlerursachen (Debugging)

1. **TOC in `<text:p>` verschachtelt** → Schema ungültig.
2. **`index-body` fehlt** → TOC bleibt leer, bis Nutzer aktualisiert.
3. **`source` und `body` vertauscht** → ungültige Reihenfolge.
4. **Entry-Template fehlt für eine Ebene** → Einträge dieser Ebene erscheinen nicht.
5. **Undefinierte Style-Namen** → Validierungs-/Darstellungsfehler.
6. **`xlink:href` ohne passenden Bookmark** → Links ins Leere, keine Seitenzahlauflösung.

---

## 9. Referenzen (Spezifikation)

- OASIS OpenDocument v1.4 (aktuell): https://docs.oasis-open.org/office/OpenDocument/v1.4/os/
- OASIS OpenDocument v1.3: https://docs.oasis-open.org/office/OpenDocument/v1.3/
- OASIS OpenDocument v1.2 (auch ISO/IEC 26300:2015): https://docs.oasis-open.org/office/v1.2/
- OASIS ODF Technical Committee: https://www.oasis-open.org/committees/tc_home.php?wg_abbrev=office

Relevante Abschnitte: "Table of Content", "Table of Content Source", "Index Body",
"Index Entry Templates" (Kapitel *Text Indices* im Schema-Teil der Spezifikation).
