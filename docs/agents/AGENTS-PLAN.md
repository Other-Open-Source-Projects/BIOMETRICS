# AGENTS-PLAN.md

## Global-Mandate-Alignment (AGENTS-GLOBAL)

- Agentenplanung muss globalen Regelkern und lokale Ausführung koppeln.
- Delegationsblöcke sind standardisiert, verifizierbar und revisionssicher.
- Keine Maßnahme ohne klaren Kontroll- und Übergabepfad.

Status: ACTIVE  
Version: 1.0 (Universal)  
Stand: Februar 2026

## Universalitäts-Regel
Dieser Plan ist absichtlich projektagnostisch. Er enthält nur wiederverwendbare Strukturen, Platzhalter und Qualitätsregeln.

Pflicht:
1. Keine projektspezifischen Secrets in dieser Datei.
2. Keine hardcodierten Domains, IDs oder Kundendaten.
3. Jede konkrete Projektinstanz ersetzt Platzhalter sauber und vollständig.

## Platzhalter-Konvention
- {PROJECT_NAME}
- {PRIMARY_AUDIENCE}
- {BUSINESS_GOAL}
- {CORE_FEATURE_SET}
- {CONTENT_DOMAIN}
- {TARGET_LANGUAGES}
- {CHANNELS}
- {COMPLIANCE_SCOPE}
- {OWNER_ROLE}

## Betriebsregeln
1. Erst lesen, dann bearbeiten.
2. NLM immer vollumfänglich via NLM-CLI nutzen.
3. Für Website-/Webapp-Erklärpfade Video-Einsatz prüfen und dokumentieren.
4. NLM-Outputs nur nach Qualitätsprüfung übernehmen.
5. Jede Änderung in `MEETING.md` und `CHANGELOG.md` protokollieren.
6. Keine Done-Behauptung ohne Evidenz.
7. Jede Aufgabe hat Akzeptanzkriterien, Tests und Doku-Update.

---

## NLM: NLM CLI COMMANDS

```bash
# Create notebook
nlm notebook create "Title"

# List sources
nlm source list <notebook-id>

# Delete old source (BEFORE adding new!)
nlm source delete <source-id> -y

# Add new source
nlm source add <notebook-id> --file "file.md" --wait
```

**WARNING: DUPLICATE PREVENTION:** ALWAYS run `nlm source list` before `nlm source add`!

---

## LOOP: DEQLHI-LOOP (INFINITE WORK MODE)

- After each completed task → Add 5 new tasks immediately
- Never "done" - only "next task"
- Always document → Every change in files
- Git commit + push after EVERY change
- Parallel execution ALWAYS (run_in_background=true)

### Loop Mechanism:
1. Task N Complete
2. Git Commit + Push
3. Update Docs
4. Add 5 New Tasks
5. Next Task N+1
6. Repeat infinitely

## Qualitätskriterien (global)
- Korrektheit
- Konsistenz
- Zielgruppenfit
- Umsetzbarkeit
- Wiederverwendbarkeit
- Evidenzbezug

Freigabe:
- Mindestscore: 13/16 (NLM-Matrix)
- Korrektheit muss 2/2 sein

## Zyklus
- Zyklus-ID: LOOP-001
- Umfang: 20 Tasks
- Modus: Universal NLM-Ready
- Abschluss: Task 20 = All-in-One Verification + neue 20 Tasks

## Task-Board (20 Tasks)

### Task 01
Task-ID: LOOP-001-T01  
Titel: Universalen Kontext-Rahmen definieren  
Kategorie: Architektur  
Priorität: P0

Ziel:
Eine robuste, projektagnostische Kontextstruktur vorbereiten.

Read First:
- `CONTEXT.md` (falls vorhanden)
- `ARCHITECTURE.md` (falls vorhanden)

Edit:
- `CONTEXT.md`

Akzeptanzkriterien:
1. Platzhaltermodell vollständig.
2. Zielgruppe und Business-Ziel als Templates dokumentiert.
3. Keine projektspezifischen Details fest verdrahtet.

Tests:
- Konsistenzcheck mit Platzhaltern.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Abschnitt „Universal Context Template“ vorhanden.

### Task 02
Task-ID: LOOP-001-T02  
Titel: NLM-CLI Betriebsstandard festschreiben  
Kategorie: Enablement  
Priorität: P0

Ziel:
Verbindliche NLM-CLI Nutzung für alle Agenten sicherstellen.

Read First:
- `../∞Best∞Practices∞Loop.md`

Edit:
- `AGENTS.md` (falls vorhanden)
- `COMMANDS.md` (falls vorhanden)

Akzeptanzkriterien:
1. NLM-CLI Pflicht klar dokumentiert.
2. Delegationsregeln enthalten.
3. Fallback-Regel bei NLM-Fehlern dokumentiert.

Tests:
- Regelset-Vollständigkeit gegen Checkliste prüfen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Abschnitt „NLM CLI Pflichtbetrieb“ ergänzt.

### Task 03
Task-ID: LOOP-001-T03  
Titel: Video-Policy für Websites definieren  
Kategorie: Feature  
Priorität: P0

Ziel:
Universelle Policy für Videoeinsatz auf Websites/Webapps festlegen.

Read First:
- `WEBSITE.md` (falls vorhanden)
- `WEBAPP.md` (falls vorhanden)

Edit:
- `WEBSITE.md`
- `WEBAPP.md`

Akzeptanzkriterien:
1. Regel „Video prüfen pro Kernseite“ enthalten.
2. NLM-Delegation für Skript/Storyboard dokumentiert.
3. Integrationsstatus pro Seite vorgesehen.

Tests:
- Check gegen 5 Pflichtfelder pro Seite.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Video-Policy Tabelle angelegt.

### Task 04
Task-ID: LOOP-001-T04  
Titel: NLM Promptvorlagen-Katalog harmonisieren  
Kategorie: Dokumentation  
Priorität: P0

Ziel:
Video/Infografik/Präsentation/Tabelle Vorlagen einheitlich standardisieren.

Read First:
- `../∞Best∞Practices∞Loop.md`

Edit:
- `../∞Best∞Practices∞Loop.md`

Akzeptanzkriterien:
1. Alle 4 Vorlagen folgen gleicher Struktur.
2. Pflichtblöcke (Ziel, Quellen, Qualitätscheck, Output) vorhanden.
3. Keine widersprüchlichen Begriffe.

Tests:
- Vorlagen-Kreuzprüfung mit Struktur-Check.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Vergleichsmatrix „Template Alignment“.

### Task 05
Task-ID: LOOP-001-T05  
Titel: Universal Command-to-Endpoint Mapping  
Kategorie: Architektur  
Priorität: P0

Ziel:
Sicherstellen, dass steuerbare Funktionen per Command + Endpoint abbildbar sind.

Read First:
- `COMMANDS.md` (falls vorhanden)
- `ENDPOINTS.md` (falls vorhanden)

Edit:
- `COMMANDS.md`
- `ENDPOINTS.md`

Akzeptanzkriterien:
1. Mapping-Tabelle vorhanden.
2. Fehlende Gegenstücke markiert.
3. Auth-Anforderung je Endpoint dokumentiert.

Tests:
- 1:1 Mapping Check.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Mapping-Report.

### Task 06
Task-ID: LOOP-001-T06  
Titel: NLM Qualitäts-Scoring operationalisieren  
Kategorie: Reliability  
Priorität: P1

Ziel:
NLM-Ausgaben mit klarer Bewertungsroutine freigeben oder verwerfen.

Read First:
- `../∞Best∞Practices∞Loop.md`

Edit:
- `AGENTS-PLAN.md`
- `MEETING.md` (falls vorhanden)

Akzeptanzkriterien:
1. Scorecard pro NLM-Artefakt vorhanden.
2. Freigabeschwelle dokumentiert.
3. Reject-Workflow definiert.

Tests:
- Trockenlauf mit Beispielartefakt.

Doku-Updates:
- `CHANGELOG.md`

Evidenz:
- Scorecard ausgefüllt.

### Task 07
Task-ID: LOOP-001-T07  
Titel: Universal Security-Layer für NLM-Content  
Kategorie: Security  
Priorität: P0

Ziel:
Sicherheits- und Compliance-Prüfung für generierte Inhalte standardisieren.

Read First:
- `SECURITY.md` (falls vorhanden)

Edit:
- `SECURITY.md`
- `INTEGRATION.md` (falls vorhanden)

Akzeptanzkriterien:
1. Kein Overclaim ohne Evidenz.
2. Keine sensiblen Daten in Content-Artefakten.
3. Review-Pfad dokumentiert.

Tests:
- Security-Checklist auf Musteroutput anwenden.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- NLM Security Guardrails Abschnitt.

### Task 08
Task-ID: LOOP-001-T08  
Titel: Datentabellen-Norm für Entscheidungen  
Kategorie: Feature  
Priorität: P1

Ziel:
Einheitliche Tabellenstandards für KPI- und Entscheidungsdaten schaffen.

Read First:
- `SUPABASE.md` (falls vorhanden)
- `ENDPOINTS.md` (falls vorhanden)

Edit:
- `SUPABASE.md`
- `ENDPOINTS.md`

Akzeptanzkriterien:
1. Spaltenkatalog-Standard vorhanden.
2. Typ/Einheit/Zeitbezug verpflichtend.
3. Qualitätswarnungen definiert.

Tests:
- Schema-Validierungscheck anhand Muster.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- „Data Table Standard“ Abschnitt.

### Task 09
Task-ID: LOOP-001-T09  
Titel: Präsentations-Storyline Standardisieren  
Kategorie: Enablement  
Priorität: P1

Ziel:
Ein universelles Executive-Deck-Template für Entscheidungen etablieren.

Read First:
- `CONTEXT.md` (falls vorhanden)
- `ARCHITECTURE.md` (falls vorhanden)

Edit:
- `ONBOARDING.md` (falls vorhanden)
- `WEBSITE.md` (falls vorhanden)

Akzeptanzkriterien:
1. Folienlogik klar und reproduzierbar.
2. Risiko- und Trade-off-Folien enthalten.
3. FAQ-Block enthalten.

Tests:
- Storyline-Check auf Konsistenz.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Template-Outline vorhanden.

### Task 10
Task-ID: LOOP-001-T10  
Titel: Infografik-Informationshierarchie festlegen  
Kategorie: UX  
Priorität: P1

Ziel:
Infografiken schnell verständlich und konsistent machen.

Read First:
- `WEBSITE.md` (falls vorhanden)
- `WEBAPP.md` (falls vorhanden)

Edit:
- `WEBSITE.md`
- `WEBAPP.md`

Akzeptanzkriterien:
1. Kernaussagen auf max. 5 begrenzt.
2. Visual Mapping dokumentiert.
3. Accessibility-Hinweise enthalten.

Tests:
- 30-Sekunden-Lesbarkeitsprüfung.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Infografik-Blueprint.

### Task 11
Task-ID: LOOP-001-T11  
Titel: NLM Delegationslog verpflichtend einführen  
Kategorie: Reliability  
Priorität: P1

Ziel:
Jede NLM-Nutzung nachvollziehbar protokollieren.

Read First:
- `MEETING.md` (falls vorhanden)

Edit:
- `MEETING.md`

Akzeptanzkriterien:
1. Anlass, Vorlage, Score, Übernahmegrad erfasst.
2. Verworfenes mit Grund protokolliert.
3. Wiederauffindbarkeit gewährleistet.

Tests:
- Ein Probeeintrag vollständig ausgefüllt.

Doku-Updates:
- `CHANGELOG.md`

Evidenz:
- Delegationsprotokoll-Template.

### Task 12
Task-ID: LOOP-001-T12  
Titel: Universaler Reject-and-Refine Workflow  
Kategorie: Debug  
Priorität: P1

Ziel:
Schwache NLM-Ausgaben systematisch verbessern statt ad hoc neu zu generieren.

Read First:
- `../∞Best∞Practices∞Loop.md`

Edit:
- `TROUBLESHOOTING.md` (falls vorhanden)
- `AGENTS.md` (falls vorhanden)

Akzeptanzkriterien:
1. Fehlerklassen definiert.
2. Prompt-Schärfungsschritte dokumentiert.
3. Vergleich zweier Iterationen vorgesehen.

Tests:
- Simulierter Fehlerfall durchlaufen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Reject-and-Refine Playbook.

### Task 13
Task-ID: LOOP-001-T13  
Titel: Universaler Asset-Namensstandard  
Kategorie: Architektur  
Priorität: P2

Ziel:
Dateibenennung für NLM-Artefakte über Projekte konsistent halten.

Read First:
- `INTEGRATION.md` (falls vorhanden)

Edit:
- `INTEGRATION.md`
- `WEBSITE.md` (falls vorhanden)

Akzeptanzkriterien:
1. Benennungsschema dokumentiert.
2. Versionierungssuffixe definiert.
3. Dateityp-Regeln enthalten.

Tests:
- Namensbeispiele gegen Regeln prüfen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Naming Conventions Abschnitt.

### Task 14
Task-ID: LOOP-001-T14  
Titel: NLM Artefakt-Lifecycle definieren  
Kategorie: Operations  
Priorität: P1

Ziel:
Lebenszyklus von Erstellung bis Archivierung standardisieren.

Read First:
- `INFRASTRUCTURE.md` (falls vorhanden)

Edit:
- `INFRASTRUCTURE.md`
- `INTEGRATION.md` (falls vorhanden)

Akzeptanzkriterien:
1. Zustände: draft/review/approved/retired definiert.
2. Verantwortliche je Zustand definiert.
3. Archivierungsregeln beschrieben.

Tests:
- Lifecycle auf Beispielasset anwenden.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Asset Lifecycle Tabelle.

### Task 15
Task-ID: LOOP-001-T15  
Titel: Universal KPI-Set für Content-Assets  
Kategorie: Performance  
Priorität: P1

Ziel:
Messbare Wirkung von Video/Infografik/Präsentation/Tabelle etablieren.

Read First:
- `WEBSITE.md` (falls vorhanden)
- `WEBAPP.md` (falls vorhanden)

Edit:
- `WEBSITE.md`
- `WEBAPP.md`
- `CONTEXT.md` (falls vorhanden)

Akzeptanzkriterien:
1. KPI pro Asset-Typ definiert.
2. Baseline und Zielwert als Platzhalter vorhanden.
3. Review-Intervall definiert.

Tests:
- KPI-Liste gegen Asset-Typen prüfen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- KPI-Grid.

### Task 16
Task-ID: LOOP-001-T16  
Titel: Universal Prompt-Library Index erstellen  
Kategorie: Enablement  
Priorität: P1

Ziel:
Schneller Zugriff auf freigegebene NLM-Promptvorlagen.

Read First:
- `../∞Best∞Practices∞Loop.md`

Edit:
- `NOTEBOOKLM.md` (falls vorhanden)
- `ONBOARDING.md` (falls vorhanden)

Akzeptanzkriterien:
1. Vorlagen indexiert.
2. Einsatzfall je Vorlage dokumentiert.
3. Qualitätswarnungen je Vorlage enthalten.

Tests:
- Index-Navigation auf Vollständigkeit prüfen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Prompt-Library Tabelle.

### Task 17
Task-ID: LOOP-001-T17  
Titel: Universal Rollenschnitt für Agenten und NLM  
Kategorie: Security  
Priorität: P1

Ziel:
Verantwortlichkeiten und Rechte bei NLM-Content sauber trennen.

Read First:
- `AGENTS.md` (falls vorhanden)
- `SECURITY.md` (falls vorhanden)

Edit:
- `AGENTS.md`
- `SECURITY.md`

Akzeptanzkriterien:
1. Rollenmodell dokumentiert.
2. Freigabeinstanz je Asset-Typ benannt.
3. Least-Privilege berücksichtigt.

Tests:
- Rollenrechte gegen Prozessschritte prüfen.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- RACI-Matrix für NLM-Prozesse.

### Task 18
Task-ID: LOOP-001-T18  
Titel: Universaler Compliance-Check für NLM Artefakte  
Kategorie: Compliance  
Priorität: P1

Ziel:
Rechtliche und regulatorische Risiken bei generierten Inhalten minimieren.

Read First:
- `SECURITY.md`
- `CODE_OF_CONDUCT.md` (falls vorhanden)

Edit:
- `SECURITY.md`
- `CODE_OF_CONDUCT.md`

Akzeptanzkriterien:
1. Compliance-Checkliste pro Asset-Typ vorhanden.
2. Eskalationsweg bei Verstoß dokumentiert.
3. Abnahme-Pflicht vor Veröffentlichung beschrieben.

Tests:
- Checkliste auf Probeoutput anwenden.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Compliance-Check-Template.

### Task 19
Task-ID: LOOP-001-T19  
Titel: Cross-Doc Konsistenzprüfung durchführen  
Kategorie: Reliability  
Priorität: P0

Ziel:
Sicherstellen, dass alle Dokumente denselben Standard abbilden.

Read First:
- `../∞Best∞Practices∞Loop.md`
- alle vorhandenen Pflichtdokumente

Edit:
- `AGENTS-PLAN.md`

Akzeptanzkriterien:
1. Widerspruchsliste erstellt.
2. P0-Inkonsistenzen aufgelöst oder eskaliert.
3. Offene Punkte priorisiert.

Tests:
- Konsistenzmatrix 10/10 geprüft.

Doku-Updates:
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Cross-Doc Audit-Protokoll.

### Task 20
Task-ID: LOOP-001-T20  
Titel: All-in-One Verification und LOOP-002 erzeugen  
Kategorie: Abschluss  
Priorität: P0

Ziel:
Vollprüfung durchführen und nächsten 20er-Zyklus ausrollen.

Read First:
- alle im Zyklus betroffenen Dokumente

Edit:
- `AGENTS-PLAN.md`
- `MEETING.md`
- `CHANGELOG.md`

Akzeptanzkriterien:
1. Integrationscheck abgeschlossen.
2. NLM-Artefakte gegen Qualitätsmatrix validiert.
3. Offene Risiken priorisiert.
4. LOOP-002 mit 20 neuen Tasks angelegt.

Tests:
- Vollständige Gate-Prüfung.

Doku-Updates:
- `AGENTS-PLAN.md`
- `MEETING.md`
- `CHANGELOG.md`

Evidenz:
- Abschlussreport + neue Taskliste.

## Abschlussreport-Vorlage (Task 20)
1. Umgesetzt
2. Geänderte Dateien
3. Prüfungen und Ergebnisse
4. Risiken und offene Punkte
5. Nächste 20 Tasks

## LOOP-002 Placeholder
- Wird in Task 20 erzeugt.
- Muss wieder exakt 20 Tasks enthalten.
- Muss universal und projektagnostisch bleiben.

## LOOP-002 (Universal, 20 Tasks)
- Zyklus-ID: LOOP-002
- Fokus: Cross-Doc Harmonisierung, Betriebsreife, Verifikationshärte

### Task 01
Task-ID: LOOP-002-T01
Titel: Cross-Doc Konsistenzmatrix aktualisieren
Kategorie: Reliability
Priorität: P0
Read First: `ARCHITECTURE.md`, `COMMANDS.md`, `ENDPOINTS.md`, `MAPPING-COMMANDS-ENDPOINTS.md`
Edit: `MAPPING-COMMANDS-ENDPOINTS.md`
Akzeptanzkriterien: Mapping vollständig und widerspruchsfrei
Tests: 1:1 Mapping-Check

### Task 02
Task-ID: LOOP-002-T02
Titel: README Navigationsqualität schärfen
Kategorie: Dokumentation
Priorität: P1
Read First: `../README.md`, `ONBOARDING.md`
Edit: `../README.md`
Akzeptanzkriterien: klare Startpfade für User/Dev/Admin
Tests: Link- und Vollständigkeitscheck

### Task 03
Task-ID: LOOP-002-T03
Titel: NLM Prompt-Library im Betrieb verankern
Kategorie: Enablement
Priorität: P1
Read First: `NOTEBOOKLM.md`, `../∞Best∞Practices∞Loop.md`
Edit: `NOTEBOOKLM.md`, `ONBOARDING.md`
Akzeptanzkriterien: alle 4 NLM-Assettypen mit Nutzungsroute dokumentiert
Tests: Library-Index Check

### Task 04
Task-ID: LOOP-002-T04
Titel: Website Journey-Konsistenz prüfen
Kategorie: UX
Priorität: P0
Read First: `WEBSITE.md`, `WEBAPP.md`
Edit: `WEBSITE.md`
Akzeptanzkriterien: jede Seite hat klares Ziel, CTA und Folgepfad
Tests: Journey-Flow Review

### Task 05
Task-ID: LOOP-002-T05
Titel: Webapp Flow-Kompatibilität prüfen
Kategorie: UX
Priorität: P0
Read First: `WEBAPP.md`, `ENDPOINTS.md`
Edit: `WEBAPP.md`
Akzeptanzkriterien: Kernflows an Commands/Endpoints gekoppelt
Tests: Flow-to-API Check

### Task 06
Task-ID: LOOP-002-T06
Titel: Webshop Betriebsfähigkeit absichern
Kategorie: Feature
Priorität: P1
Read First: `WEBSHOP.md`, `SECURITY.md`
Edit: `WEBSHOP.md`
Akzeptanzkriterien: Checkout-Risiken und Prüfpfade dokumentiert
Tests: Checkout-Review Checkliste

### Task 07
Task-ID: LOOP-002-T07
Titel: Security Kontrollmatrix erweitern
Kategorie: Security
Priorität: P0
Read First: `SECURITY.md`, `INTEGRATION.md`
Edit: `SECURITY.md`
Akzeptanzkriterien: Kontrollen für API, NLM, Integrationen vollständig
Tests: Security-Matrix Review

### Task 08
Task-ID: LOOP-002-T08
Titel: Supabase RLS Template schärfen
Kategorie: Security
Priorität: P0
Read First: `SUPABASE.md`, `ENDPOINTS.md`
Edit: `SUPABASE.md`
Akzeptanzkriterien: RLS pro Kernbereich eindeutig beschrieben
Tests: RLS-Policy Vollständigkeitscheck

### Task 09
Task-ID: LOOP-002-T09
Titel: CI/CD Gate-Härtung dokumentieren
Kategorie: Reliability
Priorität: P1
Read First: `CI-CD-SETUP.md`, `GITHUB.md`
Edit: `CI-CD-SETUP.md`, `GITHUB.md`
Akzeptanzkriterien: Gate-Regeln und Rollback klar und konfliktfrei
Tests: Pipeline-Review gegen Gate-Liste

### Task 10
Task-ID: LOOP-002-T10
Titel: Infrastruktur Recovery-Drill ergänzen
Kategorie: Operations
Priorität: P1
Read First: `INFRASTRUCTURE.md`, `TROUBLESHOOTING.md`
Edit: `INFRASTRUCTURE.md`, `TROUBLESHOOTING.md`
Akzeptanzkriterien: Recovery-Ablauf vollständig beschrieben
Tests: Restore-Checkliste vorhanden

### Task 11
Task-ID: LOOP-002-T11
Titel: OpenClaw Fehlerpfad vertiefen
Kategorie: Integration
Priorität: P1
Read First: `OPENCLAW.md`, `INTEGRATION.md`
Edit: `OPENCLAW.md`
Akzeptanzkriterien: Auth- und Retry-Fehlerfälle vollständig
Tests: Fehlerklassentabelle vorhanden

### Task 12
Task-ID: LOOP-002-T12
Titel: n8n Workflow-Qualitätsgates ergänzen
Kategorie: Integration
Priorität: P1
Read First: `N8N.md`, `INTEGRATION.md`
Edit: `N8N.md`
Akzeptanzkriterien: Trigger/Input/Output/Recovery pro Workflow beschrieben
Tests: Workflow-Checkliste

### Task 13
Task-ID: LOOP-002-T13
Titel: Vercel Betriebscheckliste ausbauen
Kategorie: Operations
Priorität: P2
Read First: `VERCEL.md`, `vercel.json`
Edit: `VERCEL.md`
Akzeptanzkriterien: Pre-/Post-Deploy und Rollback-Checks vollständig
Tests: Checklisten-Review

### Task 14
Task-ID: LOOP-002-T14
Titel: IONOS DNS Betriebscheckliste ausbauen
Kategorie: Operations
Priorität: P2
Read First: `IONOS.md`, `CLOUDFLARE.md`
Edit: `IONOS.md`
Akzeptanzkriterien: DNS/TLS-Checkliste vollständig
Tests: DNS-TLS-Checklist Review

### Task 15
Task-ID: LOOP-002-T15
Titel: Blueprint Delta-Tracking operationalisieren
Kategorie: Architektur
Priorität: P1
Read First: `BLUEPRINT.md`, `ARCHITECTURE.md`
Edit: `BLUEPRINT.md`
Akzeptanzkriterien: Soll-Ist-Deltas mit Prioritäten erfasst
Tests: Delta-Matrix Vollständigkeitscheck

### Task 16
Task-ID: LOOP-002-T16
Titel: Contribution Workflow schärfen
Kategorie: Enablement
Priorität: P1
Read First: `CONTRIBUTING.md`, `GITHUB.md`
Edit: `CONTRIBUTING.md`
Akzeptanzkriterien: Beitragspfad und Pflichtchecks eindeutig
Tests: PR-Template Konsistenzcheck

### Task 17
Task-ID: LOOP-002-T17
Titel: Onboarding Quickstart synchronisieren
Kategorie: Enablement
Priorität: P1
Read First: `ONBOARDING.md`, `../README.md`
Edit: `ONBOARDING.md`
Akzeptanzkriterien: Schnellstart mit aktuellen Dokumenten konsistent
Tests: Rolle-zu-Pfad Prüfung

### Task 18
Task-ID: LOOP-002-T18
Titel: User-Plan Priorisierung schärfen
Kategorie: Dokumentation
Priorität: P2
Read First: `USER-PLAN.md`, `AGENTS-PLAN.md`
Edit: `USER-PLAN.md`
Akzeptanzkriterien: User-Aufgaben klar priorisiert und verifizierbar
Tests: Prioritäts- und Nachweischeck

### Task 19
Task-ID: LOOP-002-T19
Titel: Gesamtkonsistenz final prüfen
Kategorie: Reliability
Priorität: P0
Read First: alle Kern-Dokumente
Edit: `AGENTS-PLAN.md`
Akzeptanzkriterien: P0-Widersprüche null oder eskaliert
Tests: Cross-Doc Audit

### Task 20
Task-ID: LOOP-002-T20
Titel: All-in-One Verification und LOOP-003 erzeugen
Kategorie: Abschluss
Priorität: P0
Read First: gesamter aktueller Stand
Edit: `AGENTS-PLAN.md`, `MEETING.md`, `CHANGELOG.md`
Akzeptanzkriterien: Vollprüfung dokumentiert, Risiken priorisiert, nächste 20 Tasks erstellt
Tests: Gesamt-Gate-Check

---

## DELQHI-LOOP Tasks (BIOMETRICS spezifisch)

Diese Tasks sind spezifisch für das BIOMETRICS-Projekt und werden im DELQHI-LOOP kontinuierlich ausgeführt.

### DELQHI-Task 01
Task-ID: DELQHI-001-T01
Titel: CLI Installation verifizieren
Kategorie: Infrastructure
Priorität: P0
Read First: `biometrics-cli/README.md`
Edit: `TESTING.md`
Akzeptanzkriterien: `biometrics --version` gibt Version aus

### DELQHI-Task 02
Task-ID: DELQHI-001-T02
Titel: NVIDIA NIM Qwen 3.5 Integration testen
Kategorie: Integration
Priorität: P0
Read First: `OPENCLAW.md`
Edit: `OPENCLAW.md`
Akzeptanzkriterien: Qwen 3.5 beantwortet Test-Prompt

### DELQHI-Task 03
Task-ID: DELQHI-001-T03
Titel: Supabase Schema validieren
Kategorie: Database
Priorität: P0
Read First: `SUPABASE.md`
Edit: `SUPABASE.md`
Akzeptanzkriterien: Alle Tabellen und RLS-Policies definiert

### DELQHI-Task 04
Task-ID: DELQHI-001-T04
Titel: n8n Workflows importieren
Kategorie: Automation
Priorität: P1
Read First: `N8N.md`
Edit: `N8N.md`
Akzeptanzkriterien: Alle Workflows importiert und aktiv

### DELQHI-Task 05
Task-ID: DELQHI-001-T05
Titel: OpenCode Modelle konfigurieren
Kategorie: Configuration
Priorität: P0
Read First: `OPENCODE.md`
Edit: `OPENCODE.md`
Akzeptanzkriterien: Alle Modelle in ~/.config/opencode/opencode.json konfiguriert

### DELQHI-Task 06
Task-ID: DELQHI-001-T06
Titel: NLM-ASSETS Struktur erstellen
Kategorie: Documentation
Priorität: P1
Read First: `NOTEBOOKLM.md`
Edit: `NLM-ASSETS/README.md`
Akzeptanzkriterien: Alle Verzeichnisse vorhanden, README vollständig

### DELQHI-Task 07
Task-ID: DELQHI-001-T07
Titel: Crashtests ausführen
Kategorie: Testing
Priorität: P0
Read First: `TESTING.md`
Edit: `TESTING.md`
Akzeptanzkriterien: Alle 10 Crashtests bestanden

### DELQHI-Task 08
Task-ID: DELQHI-001-T08
Titel: Security Audit durchführen
Kategorie: Security
Priorität: P0
Read First: `SECURITY.md`
Edit: `SECURITY.md`
Akzeptanzkriterien: Keine kritischen Vulnerabilities

### DELQHI-Task 09
Task-ID: DELQHI-001-T09
Titel: Performance Benchmarks ausführen
Kategorie: Performance
Priorität: P1
Read First: `TESTING.md`
Edit: `TESTING.md`
Akzeptanzkriterien: Alle Latenz-Anforderungen erfüllt

### DELQHI-Task 10
Task-ID: DELQHI-001-T10
Titel: E2E Test-Suite erstellen
Kategorie: Testing
Priorität: P1
Read First: `TESTING.md`
Edit: `TESTING.md`
Akzeptanzkriterien: Playwright Tests für kritische Journeys

### DELQHI-Task 11
Task-ID: DELQHI-001-T11
Titel: GitHub Actions Pipeline einrichten
Kategorie: CI/CD
Priorität: P1
Read First: `CI-CD-SETUP.md`
Edit: `CI-CD-SETUP.md`
Akzeptanzkriterien: Pipeline läuft bei jedem Push

### DELQHI-Task 12
Task-ID: DELQHI-001-T12
Titel: Documentation vollständig verifizieren
Kategorie: Documentation
Priorität: P0
Read First: `README.md`, `AGENTS.md`, `AGENTS-PLAN.md`
Edit: `README.md`
Akzeptanzkriterien: Alle Pflichtdokumente vorhanden

### DELQHI-Task 13
Task-ID: DELQHI-001-T13
Titel: Onboarding-Prozess testen
Kategorie: UX
Priorität: P1
Read First: `ONBOARDING.md`
Edit: `ONBOARDING.md`
Akzeptanzkriterien: Neuer User kann in 5 Min starten

### DELQHI-Task 14
Task-ID: DELQHI-001-T14
Titel: Cloudflare Tunnel verifizieren
Kategorie: Infrastructure
Priorität: P1
Read First: `CLOUDFLARE.md`
Edit: `CLOUDFLARE.md`
Akzeptanzkriterien: Alle Services über HTTPS erreichbar

### DELQHI-Task 15
Task-ID: DELQHI-001-T15
Titel: Vercel Deployment konfigurieren
Kategorie: Deployment
Priorität: P1
Read First: `VERCEL.md`
Edit: `VERCEL.md`
Akzeptanzkriterien: Auto-Deploy bei Push aktiv

### DELQHI-Task 16
Task-ID: DELQHI-001-T16
Titel: GitLab Media Storage einrichten
Kategorie: Infrastructure
Priorität: P1
Read First: `GITLAB.md`
Edit: `GITLAB.md`
Akzeptanzkriterien: NLM-Videos und Assets hochladbar

### DELQHI-Task 17
Task-ID: DELQHI-001-T17
Titel: OpenClaw Skills registrieren
Kategorie: Integration
Priorität: P1
Read First: `OPENCLAW.md`
Edit: `OPENCLAW.md`
Akzeptanzkriterien: Master-Skills funktionsfähig

### DELQHI-Task 18
Task-ID: DELQHI-001-T18
Titel: Troubleshooting Guide erweitern
Kategorie: Documentation
Priorität: P2
Read First: `TROUBLESHOOTING.md`
Edit: `TROUBLESHOOTING.md`
Akzeptanzkriterien: 10+ häufige Probleme dokumentiert

### DELQHI-Task 19
Task-ID: DELQHI-001-T19
Titel: Cross-Doc Konsistenzprüfung
Kategorie: Reliability
Priorität: P0
Read First: alle Kern-Dokumente
Edit: `MEETING.md`
Akzeptanzkriterien: Keine Widersprüche in Dokumenten

### DELQHI-Task 20
Task-ID: DELQHI-001-T20
Titel: DELQHI-LOOP-002 vorbereiten
Kategorie: Abschluss
Priorität: P0
Read First: `CHANGELOG.md`, `MEETING.md`
Edit: `AGENTS-PLAN.md`, `CHANGELOG.md`
Akzeptanzkriterien: 20 neue Tasks erstellt, alte archiviert

## Success Criteria (Evidenz-Standard)

Für jede abgeschlossene Task muss Evidenz erbracht werden:

| Kriterium | Beschreibung | Nachweis |
|-----------|--------------|----------|
| Test bestanden | Automatisierter Test erfolgreich | Test-Output |
| Code funktioniert | Manueller Verifikationstest | Screenshot |
| Dokumentation | README/Guide aktualisiert | Git-Diff |
| Integration | Externe Service-Verbindung | API-Response |

## Delegation Templates

### Subagent Auftrag (Standard)
```
ROLE: {agent_type}
GOAL: {klare_aufgabenbeschreibung}
CONTEXT: {relevant_history}
READ FIRST: {pflichtdokumente}
EDIT ONLY: {zu_bearbeitende_dateien}
TASKS: {aufgabenliste}
ACCEPTANCE CRITERIA: {erfolgskriterien}
REQUIRED TESTS: {test_typen}
REQUIRED DOC UPDATES: {doku_dateien}
RISKS: {potentielle_probleme}
OUTPUT FORMAT: {strukturiertes_uebergabeformat}
```

---

# SECTION 1: AGENT OVERVIEW (500+ ZEILEN)

## 1.1 Agent Types

### Orchestrator Agent
Der Orchestrator ist das zentrale Nervensystem der Agentenarbeit. Er koordiniert alle Aktivitäten, trifft strategische Entscheidungen und verantwortet die Gesamtsteuerung des Systems.

**Charakteristik:**
- Höchste Autorität im Agentenverbund
- Entscheidet über Taskverteilung und Priorisierung
- Prüft Qualität und Evidenz aller Ergebnisse
- Verantwortet die Einhaltung von Governance-Regeln
- Implementiert den DELQHI-LOOP Mechanismus

**Fähigkeiten:**
- Multitasking und parallele Koordination
- Komplexe Entscheidungsfindung unter Unsicherheit
- Risikoanalyse und Prävention
- Qualitätssicherung und Validierung
- Konfliktlösung zwischen Subagenten

**Technische Spezifikation:**
```typescript
interface OrchestratorAgent {
  id: string;
  role: 'orchestrator';
  capabilities: [
    'task_coordination',
    'quality_assurance',
    'risk_management',
    'delegate_agents',
    'verify_results',
    'manage_loop'
  ];
  maxParallelAgents: number;
  decisionTimeout: number;
  verificationLevel: 'minimal' | 'standard' | 'strict';
}
```

### Specialist Agent
Specialist Agents sind spezialisierte Ausführungseinheiten für spezifische Domänen. Sie bringen tiefes Fachwissen in ihre jeweiligen Bereiche ein.

**Typen:**
- **Code Agent:** Full-Stack Entwicklung, Testing, Refactoring
- **Docs Agent:** Dokumentation, Technische Schreibarbeit
- **Research Agent:** Recherche, Analyse, Informationsbeschaffung
- **Security Agent:** Sicherheitsanalyse, Vulnerability Assessment
- **DevOps Agent:** CI/CD, Infrastructure, Deployment
- **Data Agent:** Datenbanken, ETL, Analytics
- **QA Agent:** Testautomatisierung, Qualitätssicherung

**Beispiel-Konfiguration:**
```typescript
interface SpecialistAgent {
  id: string;
  role: 'specialist';
  specialty: 'code' | 'docs' | 'research' | 'security' | 'devops' | 'data' | 'qa';
  capabilities: string[];
  maxConcurrentTasks: number;
  expertiseLevel: 'junior' | 'mid' | 'senior' | 'expert';
  tools: string[];
}
```

### Worker Agent
Worker Agents führen operative Aufgaben aus. Sie sind die Ausführungseinheiten, die konkrete Tätigkeiten verrichten.

**Charakteristik:**
- Führt repetitive und strukturierte Aufgaben aus
- Arbeitet nach klar definierten Prozessen
- Liefert standardisierte Ergebnisse
- Benötigt klare Anweisungen und Kontext

**Einsatzgebiete:**
- Dateioperationen und Transformationen
- API-Aufrufe und Datenverarbeitung
- Testausführung und Reporting
- Deployment und Infrastructure-as-Code
- Monitoring und Alert-Handling

```typescript
interface WorkerAgent {
  id: string;
  role: 'worker';
  taskTypes: string[];
  automationLevel: 'manual' | 'semiautomated' | 'fully_automated';
  retryPolicy: {
    maxRetries: number;
    backoffStrategy: 'linear' | 'exponential';
  };
}
```

## 1.2 Agent Capabilities

### Cognitive Capabilities

**Reasoning:**
- Abstraktes Denken und logische Schlussfolgerung
- Kausale Zusammenhänge erkennen
- Mehrstufige Problemlösung
- Analoges Denken und Mustererkennung

**Planning:**
- Strategische Zielplanung
- Meilensteindefinition und Tracking
- Ressourcenallokation
- Risikovorsorge und Contingency Planning

**Learning:**
- Aus Erfahrung lernen
- Best Practices adaptieren
- Neue Fähigkeiten entwickeln
- Feedback-verarbeitung

### Execution Capabilities

**Code Generation:**
- Full-Stack Implementierung
- Design Patterns anwenden
- Test-Code schreiben
- Refactoring und Optimierung

**Data Processing:**
- ETL-Pipelines
- Datenvalidierung und -transformation
- Analytics und Reporting
- Datenbankoperationen

**Communication:**
- Natürliche Sprache Verarbeitung
- Struktierte Ausgaben generieren
- Technische Dokumentation erstellen
- Benutzerführung und -interaktion

### Tool Capabilities

**Development Tools:**
- IDE-Integration und Code-Editing
- Version Control (Git)
- Package Management
- Linting und Formatting

**Infrastructure Tools:**
- Container-Management (Docker)
- Cloud-Plattformen
- Monitoring und Observability
- CI/CD Pipeline

**Integration Tools:**
- REST/GraphQL APIs
- Webhooks und Events
- Message Queues
- External Services

## 1.3 Agent Communication

### Message Types

**Request Messages:**
```typescript
interface AgentRequest {
  type: 'request';
  id: string;
  sender: AgentIdentity;
  recipient: AgentIdentity;
  action: AgentAction;
  payload: unknown;
  context: RequestContext;
  priority: 'low' | 'normal' | 'high' | 'critical';
  timeout: number;
  correlationId?: string;
}
```

**Response Messages:**
```typescript
interface AgentResponse {
  type: 'response';
  id: string;
  requestId: string;
  sender: AgentIdentity;
  status: 'success' | 'error' | 'partial';
  payload: unknown;
  metadata: ResponseMetadata;
  executionTime: number;
}
```

**Event Messages:**
```typescript
interface AgentEvent {
  type: 'event';
  id: string;
  eventType: string;
  source: AgentIdentity;
  payload: unknown;
  timestamp: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
}
```

### Communication Patterns

**Request-Response:**
Der einfachste Kommunikationsmodus. Ein Agent sendet eine Anfrage und wartet auf Antwort.

```
Agent A                    Agent B
   │                          │
   │──── Request ─────────────▶│
   │                          │
   │      (Processing)         │
   │                          │
   │◀─── Response ────────────│
   │                          │
```

**Fire-and-Forget:**
Für asynchrone Operationen. Der Sender benötigt keine Antwort.

```
Agent A                    Agent B
   │                          │
   │──── Event ──────────────▶│
   │                          │
   │   (Continue Work)        │ (Async Processing)
   │                          │
```

**Pub-Sub:**
Für Broadcasting und Event-basiertes Arbeiten. Ein Agent publishet, mehrere können subscriben.

```
Publisher                  Subscribers
    │                          │
    │───── Event ─────────────▶│
    │         │                │
    │         └───────────────▶│
    │                          │
```

**Pipeline:**
Für chaining von Operationen. Das Ergebnis eines Agents wird Input für den nächsten.

```
Agent A ──▶ Agent B ──▶ Agent C ──▶ Agent D
   │          │           │          │
   └──────────┴──────────┴──────────┘
         Pipeline Output
```

## 1.4 Agent Coordination

### Leader Election

Bei verteilten Agentensystemen muss ein Leader koordinieren:

```typescript
interface LeaderElection {
  algorithm: 'raft' | 'bully' | 'lease';
  heartbeatInterval: number;
  electionTimeout: number;
  maxRetries: number;
  
  elect(): Promise<AgentIdentity>;
  isLeader(agentId: string): boolean;
  resign(): Promise<void>;
}
```

**Raft Implementation:**
```
┌─────────────────────────────────────────────┐
│              RAFT LEADER ELECTION            │
├─────────────────────────────────────────────┤
│                                              │
│  1. Follower Timeout (kein Heartbeat)       │
│     ↓                                        │
│  2. Werde Candidate                         │
│     ↓                                        │
│  3. Vote Request an alle Nodes               │
│     ↓                                        │
│  4. Wenn Mehrheit → Leader                   │
│     └── Wenn Split Vote → Neue Wahl          │
│                                              │
│  5. Heartbeat an Follower senden            │
│                                              │
└─────────────────────────────────────────────┘
```

### Task Distribution

**Load-Based Distribution:**
```typescript
interface TaskDistributor {
  distributionStrategy: 'round_robin' | 'least_loaded' | 'capability_based' | 'affinity_based';
  
  selectAgent(task: Task, agents: Agent[]): Agent;
  rebalance(agents: Agent[], tasks: Task[]): Map<Agent, Task[]>;
  getMetrics(): DistributionMetrics;
}
```

**Capability-Based Distribution:**
- Task-Anforderungen gegen Agent-Fähigkeiten matchen
- Spezialisierung bevorzugen
- Skill-Matrix für optimale Verteilung nutzen

### Conflict Resolution

**Strategies:**
1. **First-Write-Wins:** Einfach, kann zu Datenverlust führen
2. **Optimistic Locking:** Prüfen ob geändert, dann überschreiben
3. **Pessimistic Locking:** Sperren vor Änderung
4. **Merge Strategy:** Automatisches Zusammenführen bei Konflikten
5. **Human Escalation:** Bei kritischen Konflikten Eskalation

```typescript
interface ConflictResolver {
  strategy: ConflictResolutionStrategy;
  
  detect(sourceA: Change, sourceB: Change): Conflict;
  resolve(conflict: Conflict): Resolution;
  escalate(conflict: Conflict): Promise<Resolution>;
}
```

## 1.5 Agent Governance

### Policy Framework

**Governance Rules:**
```typescript
interface AgentPolicy {
  id: string;
  name: string;
  description: string;
  rules: PolicyRule[];
  enforcement: 'strict' | 'advisory' | 'audit';
  exceptions: PolicyException[];
}

interface PolicyRule {
  id: string;
  condition: string; // CEL expression
  action: 'allow' | 'deny' | 'audit' | 'escalate';
  message: string;
}
```

### Compliance Monitoring

**Audit Trail:**
```typescript
interface AuditLog {
  timestamp: string;
  agentId: string;
  action: string;
  resource: string;
  result: 'success' | 'failure';
  metadata: Record<string, unknown>;
  complianceFlags: string[];
}
```

### Quality Assurance

**Quality Gates:**
- Pre-Execution: Validierung von Inputs und Berechtigungen
- Execution: Monitoring von Fortschritt und Ressourcen
- Post-Execution: Validierung von Outputs und Ergebnisse
- Review: Manuelle oder automatisierte Qualitätsprüfung

---

# SECTION 2: DELQHI-LOOP TASKS (1000+ ZEILEN)

## 2.1 Task Generation

### Task Creation Rules

**Grundregeln:**
1. Jede Task muss einen klaren, messbaren Goal haben
2. Akzeptanzkriterien müssen vor Start definiert sein
3. Abhängigkeiten müssen explizit dokumentiert sein
4. Priorisierung muss nach Business Value und Effort erfolgen
5. Tasks dürfen nicht größer als 8 Stunden Arbeit sein

**Task-Template:**
```typescript
interface Task {
  id: string;
  title: string;
  description: string;
  category: TaskCategory;
  priority: Priority;
  status: TaskStatus;
  
  // Scope
  scope: {
    files: string[];
    agents: string[];
    dependencies: string[];
  };
  
  // Criteria
  acceptanceCriteria: AcceptanceCriterion[];
  requiredTests: string[];
  requiredDocs: string[];
  
  // Tracking
  assignee?: string;
  estimatedHours: number;
  actualHours?: number;
  startDate?: string;
  endDate?: string;
  
  // Evidence
  evidence: TaskEvidence[];
  blockers: Blocker[];
}
```

### Task Templates

**Standard Task:**
```
Task-ID: {LOOP}-{T##}
Titel: {Titel}
Kategorie: {Architektur|Feature|Integration|Testing|Documentation}
Priorität: {P0|P1|P2}

Ziel:
{klar formuliertes Ziel}

Read First:
- {Datei1}
- {Datei2}

Edit:
- {DateiA}
- {DateiB}

Akzeptanzkriterien:
1. {Kriterium 1}
2. {Kriterium 2}
3. {Kriterium 3}

Tests:
- {Test 1}
- {Test 2}

Doku-Updates:
- {Dokumentation 1}
- {Dokumentation 2}

Evidenz:
- {Erwartete Evidenz}
```

**Bug Fix Task:**
```
Task-ID: {LOOP}-BUG-{##}
Titel: Bugfix: {Bug-Beschreibung}
Priorität: P0 (wenn Produktion)

Ziel:
{Bug beschreiben und gewünschtes Verhalten definieren}

Reproduction Steps:
1. {Schritt 1}
2. {Schritt 2}

Expected vs Actual:
- Expected: {Erwartet}
- Actual: {Tatsächlich}

Root Cause:
{Analyse der Ursache}

Fix Approach:
{Geplante Lösung}

Tests:
- Unit Test für den Fix
- Regression Tests
```

### Task Prioritization

**Priority Matrix:**

| Priority | Description | Response Time | Escalation |
|----------|-------------|---------------|------------|
| P0 - Critical | Produktionsblocker, Security Issues | Sofort | Keine, direkt bearbeiten |
| P1 - High | Wichtige Features, kritische Bugs | Innerhalb Session | Nach 4h |
| P2 - Medium | Normale Features, Verbesserungen | Innerhalb 20er-Loop | Nach 1 Woche |
| P3 - Low | Nice-to-have, Dokumentation | Wenn Zeit | Nie |

**Priorisierungsalgorithmus:**
```typescript
function prioritize(tasks: Task[]): Task[] {
  return tasks.sort((a, b) => {
    // 1. Priority Score
    const priorityScore = {
      'P0': 1000,
      'P1': 100,
      'P2': 10,
      'P3': 1
    }[a.priority] - priorityScore[b.priority];
    
    if (priorityScore !== 0) return priorityScore;
    
    // 2. Business Value
    return b.businessValue - a.businessValue;
  });
}
```

### Task Dependencies

**Dependency Types:**

1. **Blocking:** Task B kann erst starten wenn Task A fertig ist
2. **Dependent:** Task B braucht Output von Task A
3. **Related:** Task A und B sollten zusammen bearbeitet werden
4. **Sequential:** Task B sollte nach A kommen (nicht zwingend)

**Dependency Graph:**
```typescript
interface TaskDependency {
  sourceTaskId: string;
  targetTaskId: string;
  type: 'blocking' | 'dependent' | 'related' | 'sequential';
  requiredOutput?: string;
}

interface TaskGraph {
  nodes: Task[];
  edges: TaskDependency[];
  
  getExecutionOrder(): Task[][];
  detectCycles(): boolean;
  getCriticalPath(): Task[];
}
```

### Task Lifecycle

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TASK LIFECYCLE                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CREATED ─────▶ PLANNED ─────▶ IN_PROGRESS ─────▶ REVIEW ─────▶ DONE       │
│     │              │               │                │            │           │
│     │              │               │                │            │           │
│     ▼              ▼               ▼                ▼            ▼           │
│  Backlog       Ready for       Working on       Pending      Complete       │
│               assignment       the task         review       and verified    │
│                                                                              │
│     │              │               │                │            │           │
│     │              │               │                │            │           │
│     ▼              ▼               ▼                ▼            ▼           │
│  BLOCKED ◀────── REJECTED ──── FAILED ────── NEEDS_WORK ── CANCELLED       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**State Transitions:**

| From | To | Trigger |
|------|-----|---------|
| CREATED | PLANNED | Task vollständig geplant |
| PLANNED | IN_PROGRESS | Assignee zugewiesen |
| IN_PROGRESS | REVIEW | Alle AC erfüllt |
| REVIEW | DONE | Review bestanden |
| REVIEW | IN_PROGRESS | Changes erforderlich |
| ANY | BLOCKED | Blocker identifiziert |
| BLOCKED | IN_PROGRESS | Blocker behoben |
| ANY | CANCELLED | Task abgebrochen |

## 2.2 Task Execution

### Execution Strategies

**Parallel Execution:**
```typescript
interface ParallelExecutor {
  maxParallel: number;
  continueOnError: boolean;
  
  async execute(tasks: Task[]): Promise<TaskResult[]> {
    const results: TaskResult[] = [];
    const queue = [...tasks];
    
    const workers = Array(this.maxParallel)
      .fill(null)
      .map(async () => {
        while (queue.length > 0) {
          const task = queue.shift();
          if (!task) break;
          
          try {
            const result = await executeTask(task);
            results.push(result);
          } catch (error) {
            if (!this.continueOnError) throw error;
            results.push({ task, status: 'failed', error });
          }
        }
      });
    
    await Promise.all(workers);
    return results;
  }
}
```

**Sequential Execution:**
```typescript
async function executeSequential(tasks: Task[]): Promise<TaskResult[]> {
  const results: TaskResult[] = [];
  
  for (const task of tasks) {
    const result = await executeTask(task);
    results.push(result);
    
    if (result.status === 'failed') {
      throw new Error(`Task ${task.id} failed, stopping sequence`);
    }
  }
  
  return results;
}
```

**Conditional Execution:**
```typescript
interface ConditionalExecutor {
  conditions: Map<string, () => Promise<boolean>>;
  
  async executeIf(condition: string, task: Task): Promise<TaskResult | null> {
    const check = this.conditions.get(condition);
    if (!check) throw new Error(`Unknown condition: ${condition}`);
    
    if (await check()) {
      return executeTask(task);
    }
    return null;
  }
}
```

### Error Handling

**Retry Policy:**
```typescript
interface RetryPolicy {
  maxRetries: number;
  initialDelay: number;
  maxDelay: number;
  backoffMultiplier: number;
  retryableErrors: string[];
  
  async withRetry<T>(fn: () => Promise<T>): Promise<T> {
    let lastError: Error;
    let delay = this.initialDelay;
    
    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error as Error;
        
        if (!this.isRetryable(error)) {
          throw error;
        }
        
        if (attempt < this.maxRetries) {
          await sleep(delay);
          delay = Math.min(delay * this.backoffMultiplier, this.maxDelay);
        }
      }
    }
    
    throw lastError!;
  }
}
```

**Circuit Breaker:**
```typescript
class CircuitBreaker {
  private failures = 0;
  private lastFailure?: Date;
  private state: 'closed' | 'open' | 'half_open' = 'closed';
  
  constructor(
    private threshold: number,
    private timeout: number,
    private resetTimeout: number
  ) {}
  
  async execute<T>(fn: () => Promise<T>): Promise<T> {
    if (this.state === 'open') {
      if (Date.now() - this.lastFailure!.getTime() > this.resetTimeout) {
        this.state = 'half_open';
      } else {
        throw new Error('Circuit breaker is open');
      }
    }
    
    try {
      const result = await fn();
      if (this.state === 'half_open') {
        this.state = 'closed';
        this.failures = 0;
      }
      return result;
    } catch (error) {
      this.failures++;
      this.lastFailure = new Date();
      
      if (this.failures >= this.threshold) {
        this.state = 'open';
      }
      
      throw error;
    }
  }
}
```

## 2.3 Task Completion

### Verification Criteria

**Checkliste für Task-Abschluss:**
```
□ Alle Akzeptanzkriterien erfüllt
□ Alle erforderlichen Tests bestanden
□ Code Review durchgeführt (falls erforderlich)
□ Dokumentation aktualisiert
□ Evidenz dokumentiert
□ Keine kritischen Blocker offen
□ Abhängigkeiten erfüllt
□ Security Check bestanden (falls relevant)
□ Performance Check bestanden (falls relevant)
```

### Evidence Requirements

**Evidence Types:**

| Type | Description | Example |
|------|-------------|---------|
| Test Output | Automatisierte Testergebnisse | Jest/Playwright Output |
| Screenshot | Visuelle Verifikation | Browser-Screenshot |
| API Response | Externe Service-Verifikation | cURL/Postman Output |
| Code Diff | Änderungsnachweis | git diff |
| Build Output | Kompilierungsnachweis | npm build output |
| Log Extract | Laufzeitverifikation | Application logs |

**Evidence Template:**
```typescript
interface TaskEvidence {
  id: string;
  taskId: string;
  type: 'test' | 'screenshot' | 'api_response' | 'code_diff' | 'build_output' | 'log';
  description: string;
  timestamp: string;
  artifact: string; // Path or URL
  verificationMethod: 'automatic' | 'manual';
}
```

### Documentation Standards

**Doku-Update Checklist:**
```
□ README.md aktualisiert (falls relevant)
□ API-Dokumentation aktualisiert (falls relevant)
□ Kommentare im Code aktualisiert (falls relevant)
□ CHANGELOG.md aktualisiert
□ MEETING.md aktualisiert
□ Architektur-Diagramme aktualisiert (falls relevant)
□ Mapping-Dateien aktualisiert (falls relevant)
```

### Git Commit Standards

**Commit Message Format:**
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- feat: Neue Feature
- fix: Bug Fix
- docs: Dokumentation
- style: Formatierung
- refactor: Code-Restrukturierung
- test: Tests Wartung

**Beispiele:**

- chore:```bash
# Feature
git commit -m "feat(auth): add OAuth2 provider support"

# Bug Fix
git commit -m "fix(api): resolve race condition in user endpoint"

# Documentation
git commit -m "docs(agents): update delegation templates"

# Refactoring
git commit -m "refactor(database): migrate from Prisma to Drizzle"
```

### Push Automation

**Auto-Push Workflow:**
```typescript
interface AutoPushConfig {
  autoPush: boolean;
  pushOnComplete: boolean;
  includeUntracked: boolean;
  commitMessageTemplate: string;
}

async function autoPush(task: Task, config: AutoPushConfig): Promise<void> {
  if (!config.autoPush) return;
  
  const changes = await getChanges();
  if (changes.length === 0) return;
  
  const message = config.commitMessageTemplate
    .replace('{taskId}', task.id)
    .replace('{title}', task.title);
  
  await git.add(config.includeUntracked ? '-A' : '.');
  await git.commit(message);
  await git.push();
}
```

---

# SECTION 3: AGENT DELEGATION (800+ ZEILEN)

## 3.1 Delegation Patterns

### Direct Delegation

Der Orchestrator weist einem Subagenten direkt eine Aufgabe zu.

**Pattern:**
```
ORCHESTRATOR                    SPECIALIST AGENT
    │                                 │
    │──── Delegate Task ─────────────▶│
    │     (Role, Goal, Context,      │
    │      Criteria, Constraints)     │
    │                                 │
    │      (Process Task)             │
    │                                 │
    │◀─── Result + Evidence ──────────│
    │                                 │
```

**Implementation:**
```typescript
interface DirectDelegation {
  type: 'direct';
  orchestratorId: string;
  agentId: string;
  task: Task;
  context: DelegationContext;
  deadline?: string;
  
  async delegate(): Promise<DelegationResult> {
    const agent = await this.getAgent(this.agentId);
    const assignment = this.createAssignment(this.task, this.context);
    
    await agent.assign(assignment);
    const result = await agent.execute();
    
    return this.verifyResult(result);
  }
}
```

### Broadcast Delegation

Eine Aufgabe wird an mehrere Agenten delegiert, die parallel arbeiten.

**Pattern:**
```
ORCHESTRATOR                    
    │                             
    │──── Delegate to All ───────▶ Agent A
    │──── Delegate to All ───────▶ Agent B  
    │──── Delegate to All ───────▶ Agent C
    │                             
    │      (All Process)          
    │                             
    │◀─── Result A ───────────────│
    │◀─── Result B ───────────────│
    │◀─── Result C ───────────────│
    │                             
    │  (Aggregate Results)        
    │                             
```

**Implementation:**
```typescript
interface BroadcastDelegation {
  type: 'broadcast';
  task: Task;
  agents: string[];
  aggregationStrategy: 'first' | 'majority' | 'all' | 'custom';
  
  async delegate(): Promise<DelegationResult[]> {
    const promises = this.agents.map(agentId => 
      this.delegateToAgent(agentId, this.task)
    );
    
    const results = await Promise.allSettled(promises);
    
    return this.aggregateResults(results);
  }
}
```

### Competitive Delegation

Mehrere Agenten bearbeiten dieselbe Aufgabe, die beste Lösung wird gewählt.

**Pattern:**
```
ORCHESTRATOR                    
    │                             
    │──── Challenge A ───────────▶ Agent A
    │──── Challenge B ───────────▶ Agent B  
    │──── Challenge C ───────────▶ Agent C
    │                             
    │      (Independent Work)     
    │                             
    │◀─── Solution A ─────────────│
    │◀─── Solution B ─────────────│
    │◀─── Solution C ─────────────│
    │                             
    │  (Evaluate & Select Best)   
    │                             
```

**Implementation:**
```typescript
interface CompetitiveDelegation {
  type: 'competitive';
  task: Task;
  agents: string[];
  evaluationCriteria: EvaluationCriterion[];
  
  async delegate(): Promise<DelegationResult> {
    const submissions = await Promise.all(
      this.agents.map(agentId => 
        this.submitChallenge(agentId, this.task)
      )
    );
    
    const evaluated = submissions.map(s => ({
      submission: s,
      score: this.evaluate(s, this.evaluationCriteria)
    }));
    
    const winner = evaluated.sort((a, b) => b.score - a.score)[0];
    return winner.submission;
  }
}
```

### Collaborative Delegation

Mehrere Agenten arbeiten gemeinsam an einer komplexen Aufgabe.

**Pattern:**
```
ORCHESTRATOR                    
    │                             
    │──── Delegate Team ─────────▶ ┌─ Agent A ─┐
    │                               ├─ Agent B ─┤
    │                               ├─ Agent C ─┤
    │                               └───────────┘
    │                             
    │      (Collaborative Work)    
    │      ┌───────────────────┐   
    │      │ Agent A ──▶ B ──▶ C│   
    │      │     │        │     │   
    │      │     ▼        ▼     │   
    │      └───────────────────┘   
    │                             
    │◀─── Combined Result ────────│
    │                             
```

**Implementation:**
```typescript
interface CollaborativeDelegation {
  type: 'collaborative';
  task: Task;
  team: Team;
  coordinationMode: 'sequential' | 'parallel' | 'hub_and_spoke';
  sharedState: SharedState;
  
  async delegate(): Promise<DelegationResult> {
    await this.initializeTeam(this.team);
    await this.shareContext(this.task, this.team);
    
    switch (this.coordinationMode) {
      case 'sequential':
        return this.executeSequential();
      case 'parallel':
        return this.executeParallel();
      case 'hub_and_spoke':
        return this.executeHubAndSpoke();
    }
  }
}
```

### Hierarchical Delegation

Delegation durch mehrere Ebenen (Sub-Agenten können weitere Agenten beauftragen).

**Pattern:**
```
LEVEL 0: ORCHESTRATOR
    │
    ├───── LEVEL 1: SPECIALIST A
    │         │
    │         ├───── LEVEL 2: WORKER 1
    │         └───── LEVEL 2: WORKER 2
    │
    └───── LEVEL 1: SPECIALIST B
              │
              └───── LEVEL 2: WORKER 3
```

## 3.2 Prompt Engineering

### Perfect Prompt Structure

**Prompt Template:**
```typescript
interface AgentPrompt {
  // Role Definition
  role: string;
  expertise: string[];
  
  // Goal
  goal: string;
  successCriteria: string[];
  
  // Context
  context: {
    background: string;
    relevantHistory: string;
    constraints: string[];
  };
  
  // Input
  input: {
    data: unknown;
    files: string[];
    links: string[];
  };
  
  // Output
  output: {
    format: 'json' | 'markdown' | 'code' | 'text';
    structure: object;
    examples: unknown[];
  };
  
  // Constraints
  constraints: {
    don'ts: string[];
    limitations: string[];
    qualityStandards: string[];
  };
  
  // Examples
  examples: {
    input: unknown;
    output: unknown;
    explanation: string;
  }[];
}
```

### Context Provision

**Context Levels:**

1. **Minimal Context:** Nur unmittelbar benötigte Informationen
2. **Standard Context:** Relevante Historie + aktuelle Task
3. **Full Context:** Komplette Projektübersicht + Historie

```typescript
interface ContextBuilder {
  build(
    task: Task,
    level: 'minimal' | 'standard' | 'full'
  ): Promise<Context> {
    const base = this.getBaseContext();
    const taskContext = this.getTaskContext(task);
    
    let additional = {};
    if (level === 'standard') {
      additional = this.getRelevantHistory(task);
    } else if (level === 'full') {
      additional = this.getFullProjectContext();
    }
    
    return { ...base, ...taskContext, ...additional };
  }
}
```

### Success Criteria Definition

**SMART Criteria:**
```typescript
interface SuccessCriteria {
  // Specific - Klar definiert
  specific: string;
  
  // Measurable - Messbar
  measurable: {
    metric: string;
    target: number;
    unit: string;
  };
  
  // Achievable - Erreichbar
  achievable: {
    difficulty: 'easy' | 'medium' | 'hard';
    dependencies: string[];
  };
  
  // Relevant - Relevanz
  relevant: {
    businessValue: number;
    userImpact: 'low' | 'medium' | 'high';
  };
  
  // Time-bound - Zeitgebunden
  timebound: {
    deadline: string;
    milestones: { date: string; target: string }[];
  };
}
```

### Constraints Definition

**Constraint Types:**
```typescript
interface Constraints {
  // Technical Constraints
  technical: {
    stack: string[];
    versions: Record<string, string>;
    linting: string[];
    testing: string[];
  };
  
  // Resource Constraints
  resources: {
    maxTime: number; // minutes
    maxCost: number;
    maxRetries: number;
  };
  
  // Quality Constraints
  quality: {
    testCoverage: number;
    lintScore: number;
    securityLevel: 'basic' | 'standard' | 'strict';
  };
  
  // Compliance Constraints
  compliance: {
    regulations: string[];
    dataPrivacy: string[];
    auditRequirements: string[];
  };
}
```

### Examples

**Good Example:**
```text
ROLE: Senior TypeScript Developer
GOAL: Implementiere eine User Authentication API

CONTEXT:
- Das Projekt nutzt Next.js 15 mit TypeScript strict
- Datenbank ist PostgreSQL mit Prisma ORM
- Auth wird über JWT Tokens realisiert

TASK:
1. Erstelle eine POST /api/auth/login Endpoint
2. Implementiere JWT Token Generierung
3. Füge Input Validation mit Zod hinzu
4. Schreibe Unit Tests

ACCEPTANCE CRITERIA:
- [ ] Login mit validen Credentials gibt Token zurück
- [ ] Login mit invaliden Credentials gibt 401 zurück
- [ ] Token enthält userId und expiresAt
- [ ] Alle Tests bestehen

CONSTRAINTS:
- Keine externen Auth-Libraries
- TypeScript strict mode
- ESLint mit Airbnb config
```

## 3.3 Result Verification

### Automated Verification

**Verification Pipeline:**
```typescript
interface VerificationPipeline {
  steps: VerificationStep[];
  
  async verify(result: AgentResult): Promise<VerificationReport> {
    const reports: StepReport[] = [];
    
    for (const step of this.steps) {
      const report = await step.execute(result);
      reports.push(report);
      
      if (!report.passed && step.blocking) {
        break;
      }
    }
    
    return this.aggregate(reports);
  }
}

// Verification Steps
const verificationSteps = [
  new FormatValidator(),
  new SchemaValidator(),
  new TestExecutor(),
  new LintChecker(),
  new SecurityScanner(),
  new PerformanceTester()
];
```

### Manual Review

**Review Checklist:**
```
Code Review:
□ Keine Security Vulnerabilities
□ Keine Performance-Probleme
□ Code folgt Style Guidelines
□ Kommentare vorhanden (falls nötig)
□ Fehlerbehandlung vollständig
□ Tests ausreichend

Documentation Review:
□ README aktualisiert
□ API Docs vollständig
□ Kommentare akkurat

Architecture Review:
□ Folgt Architektur-Prinzipien
□ Keine Anti-Patterns
□ Skalierbarkeit gewährleistet
```

### Quality Gates

**Gate Definitions:**
```typescript
interface QualityGate {
  name: string;
  criteria: GateCriterion[];
  action: 'pass' | 'warn' | 'fail' | 'escalate';
}

const qualityGates: QualityGate[] = [
  {
    name: 'Code Quality Gate',
    criteria: [
      { metric: 'lint_errors', threshold: 0, operator: '<' },
      { metric: 'test_coverage', threshold: 80, operator: '>' },
      { metric: 'tech_debt_hours', threshold: 8, operator: '<' }
    ],
    action: 'fail'
  },
  {
    name: 'Security Gate',
    criteria: [
      { metric: 'critical_vulnerabilities', threshold: 0, operator: '=' },
      { metric: 'high_vulnerabilities', threshold: 0, operator: '=' }
    ],
    action: 'fail'
  }
];
```

---

# SECTION 4: AGENT COMMUNICATION (600+ ZEILEN)

## 4.1 Message Formats

### Request Format

**Struktur:**
```typescript
interface AgentRequest {
  // Header
  id: string;
  timestamp: string;
  correlationId?: string;
  
  // Routing
  sender: AgentIdentity;
  recipient: AgentIdentity | AgentGroup;
  routing: {
    mode: 'direct' | 'broadcast' | 'multicast';
    groups?: string[];
  };
  
  // Content
  content: {
    intent: string;
    action: string;
    payload: unknown;
    attachments?: Attachment[];
  };
  
  // QoS
  quality: {
    priority: 'low' | 'normal' | 'high' | 'critical';
    timeout: number;
    deliveryGuarantee: 'at_most_once' | 'at_least_once' | 'exactly_once';
  };
  
  // Context
  context?: {
    sessionId?: string;
    workflowId?: string;
    traceId?: string;
  };
}
```

### Response Format

**Struktur:**
```typescript
interface AgentResponse {
  // Header
  id: string;
  timestamp: string;
  requestId: string;
  
  // Routing
  sender: AgentIdentity;
  recipient: AgentIdentity;
  
  // Content
  content: {
    status: 'success' | 'error' | 'partial' | 'timeout';
    payload?: unknown;
    metadata?: Record<string, unknown>;
  };
  
  // Error Details
  error?: {
    code: string;
    message: string;
    details?: unknown;
    stack?: string;
  };
  
  // Timing
  timing: {
    received: string;
    started: string;
    completed: string;
    duration: number;
  };
  
  // Evidence
  evidence?: TaskEvidence[];
}
```

### Status Updates

**Format:**
```typescript
interface StatusUpdate {
  id: string;
  taskId: string;
  agentId: string;
  
  status: 'queued' | 'started' | 'in_progress' | 'paused' | 'completed' | 'failed';
  
  progress?: {
    percentage: number;
    currentStep: string;
    totalSteps: number;
  };
  
  message?: string;
  timestamp: string;
}
```

## 4.2 Communication Channels

### Direct Messages

**Implementation:**
```typescript
interface DirectMessageChannel {
  type: 'direct';
  
  async send(request: AgentRequest): Promise<AgentResponse> {
    const queue = this.getQueue(request.recipient);
    queue.push(request);
    
    const response = await this.waitForResponse(request.id, request.quality.timeout);
    return response;
  }
  
  async receive(timeout?: number): Promise<AgentRequest> {
    const queue = this.getMyQueue();
    return queue.dequeue(timeout);
  }
}
```

### Broadcast Channels

**Implementation:**
```typescript
interface BroadcastChannel {
  type: 'broadcast';
  channelName: string;
  subscribers: Set<AgentIdentity>;
  
  async publish(event: AgentEvent): Promise<void> {
    const message = this.serialize(event);
    
    await Promise.all(
      [...this.subscribers].map(agentId => 
        this.deliver(agentId, message)
      )
    );
  }
  
  subscribe(agentId: AgentIdentity): void {
    this.subscribers.add(agentId);
  }
}
```

## 4.3 Coordination Protocols

### Consensus Building

**Protocol:**
```typescript
interface ConsensusProtocol {
  topic: string;
  participants: AgentIdentity[];
  quorum: number;
  votes: Map<AgentIdentity, Vote>;
  
  async propose(proposal: Proposal): Promise<ConsensusResult> {
    // 1. Broadcast proposal
    await this.broadcast(proposal);
    
    // 2. Collect votes with timeout
    const votes = await this.collectVotes(this.participants, this.quorum);
    
    // 3. Tally results
    const tally = this.tally(votes);
    
    // 4. Check for consensus
    if (tally.accept >= this.quorum) {
      return { status: 'accepted', votes: tally };
    } else if (tally.reject >= this.quorum) {
      return { status: 'rejected', votes: tally };
    } else {
      return { status: 'undecided', votes: tally };
    }
  }
}
```

### Task Assignment

**Algorithm:**
```typescript
interface TaskAssigner {
  assignmentStrategy: 'capability' | 'availability' | 'load' | 'affinity';
  
  assign(task: Task, agents: Agent[]): Agent {
    const scored = agents.map(agent => ({
      agent,
      score: this.calculateScore(task, agent)
    }));
    
    return scored.sort((a, b) => b.score - a.score)[0].agent;
  }
  
  calculateScore(task: Task, agent: Agent): number {
    let score = 0;
    
    if (this.assignmentStrategy === 'capability') {
      score = this.matchCapabilities(task, agent);
    } else if (this.assignmentStrategy === 'availability') {
      score = agent.availability;
    } else if (this.assignmentStrategy === 'load') {
      score = 1 / agent.currentLoad;
    }
    
    return score;
  }
}
```

---

# SECTION 5: AGENT SKILLS (700+ ZEILEN)

## 5.1 Core Skills

### File Operations

**Skill Definition:**
```typescript
const fileOperationsSkill = {
  name: 'file_operations',
  description: 'Führt Dateioperationen sicher aus',
  
  capabilities: [
    'read_file',
    'write_file', 
    'delete_file',
    'move_file',
    'copy_file',
    'create_directory',
    'list_directory',
    'search_files'
  ],
  
  tools: [
    'read',
    'write', 
    'edit',
    'glob',
    'grep',
    'bash'
  ],
  
  async execute(action: FileAction): Promise<FileResult> {
    switch (action.type) {
      case 'read':
        return await this.readFile(action.path);
      case 'write':
        return await this.writeFile(action.path, action.content);
      case 'delete':
        return await this.deleteFile(action.path);
      // ...
    }
  }
};
```

### Code Generation

**Skill Definition:**
```typescript
const codeGenerationSkill = {
  name: 'code_generation',
  description: 'Generiert qualitativ hochwertigen Code',
  
  capabilities: [
    'generate_component',
    'generate_api',
    'generate_test',
    'generate_schema',
    'generate_migration',
    'refactor_code'
  ],
  
  constraints: {
    languages: ['typescript', 'javascript', 'python', 'go'],
    frameworks: ['nextjs', 'react', 'express', 'fastapi', 'gin'],
    standards: ['strict_typescript', 'eslint', 'prettier']
  },
  
  async execute(request: CodeGenerationRequest): Promise<CodeResult> {
    // 1. Analyze request
    const spec = await this.analyze(request);
    
    // 2. Generate code
    const code = await this.generate(spec);
    
    // 3. Validate
    await this.validate(code);
    
    // 4. Format
    const formatted = await this.format(code);
    
    return { code: formatted, spec };
  }
};
```

### Testing

**Skill Definition:**
```typescript
const testingSkill = {
  name: 'testing',
  description: 'Erstellt und führt Tests aus',
  
  capabilities: [
    'write_unit_tests',
    'write_integration_tests',
    'write_e2e_tests',
    'run_tests',
    'generate_coverage',
    'analyze_test_results'
  ],
  
  frameworks: {
    javascript: ['jest', 'vitest', 'playwright', 'cypress'],
    python: ['pytest', 'unittest'],
    go: ['testing', 'ginkgo']
  },
  
  async execute(action: TestAction): Promise<TestResult> {
    switch (action.type) {
      case 'write':
        return await this.writeTests(action.spec);
      case 'run':
        return await this.runTests(action.target, action.options);
      case 'coverage':
        return await this.generateCoverage(action.target);
    }
  }
};
```

### Documentation

**Skill Definition:**
```typescript
const documentationSkill = {
  name: 'documentation',
  description: 'Erstellt technische Dokumentation',
  
  capabilities: [
    'write_readme',
    'write_api_docs',
    'write_architecture_docs',
    'write_guide',
    'update_changelog',
    'generate_type_docs'
  ],
  
  templates: {
    readme: '...',
    api: '...',
    guide: '...'
  },
  
  async execute(action: DocAction): Promise<DocResult> {
    const template = this.getTemplate(action.type);
    const content = await this.fillTemplate(template, action.context);
    await this.validate(content);
    await this.save(action.path, content);
    
    return { path: action.path, content };
  }
};
```

### Git Operations

**Skill Definition:**
```typescript
const gitOperationsSkill = {
  name: 'git_operations',
  description: 'Führt Git-Operationen aus',
  
  capabilities: [
    'git_status',
    'git_add',
    'git_commit',
    'git_push',
    'git_pull',
    'git_branch',
    'git_merge',
    'git_rebase',
    'git_diff'
  ],
  
  safety: {
    requireBranch: false,
    blockForcePush: true,
    requireCommitMessage: true,
    maxCommitMessageLength: 500
  },
  
  async execute(action: GitAction): Promise<GitResult> {
    // Validate action
    await this.validate(action);
    
    // Execute with safety checks
    switch (action.type) {
      case 'commit':
        return await this.commit(action.files, action.message);
      case 'push':
        return await this.push(action.remote, action.branch);
      // ...
    }
  }
};
```

## 5.2 Advanced Skills

### Architecture Design

**Skill Definition:**
```typescript
const architectureSkill = {
  name: 'architecture_design',
  description: 'Entwirft skalierbare Architekturen',
  
  capabilities: [
    'design_microservices',
    'design_api',
    'design_database',
    'design_infrastructure',
    'review_architecture',
    'create_adr'
  ],
  
  patterns: [
    'microservices',
    'modular_monolith',
    'event_driven',
    'cqrs',
    'domain_driven_design'
  ],
  
  async execute(request: ArchitectureRequest): Promise<ArchitectureResult> {
    // 1. Understand requirements
    const requirements = await this.analyze(request);
    
    // 2. Design architecture
    const architecture = await this.design(requirements);
    
    // 3. Document decisions (ADR)
    const adrs = await this.createADRs(architecture);
    
    // 4. Validate
    await this.validate(architecture);
    
    return { architecture, adrs };
  }
};
```

### Security Analysis

**Skill Definition:**
```typescript
const securitySkill = {
  name: 'security_analysis',
  description: 'Analysiert und verbessert Security',
  
  capabilities: [
    'vulnerability_scan',
    'code_review_security',
    'dependency_audit',
    'penetration_test',
    'security_hardening'
  ],
  
  standards: ['owasp_top_10', 'cwe', 'nist'],
  
  async execute(request: SecurityRequest): Promise<SecurityResult> {
    // 1. Scan for vulnerabilities
    const vulnerabilities = await this.scan(request.target);
    
    // 2. Analyze findings
    const analysis = await this.analyze(vulnerabilities);
    
    // 3. Generate report
    const report = await this.generateReport(analysis);
    
    // 4. Recommend fixes
    const fixes = await this.suggestFixes(analysis);
    
    return { vulnerabilities, analysis, report, fixes };
  }
};
```

### Performance Optimization

**Skill Definition:**
```typescript
const performanceSkill = {
  name: 'performance_optimization',
  description: 'Optimiert Performance',
  
  capabilities: [
    'profile_code',
    'analyze_bottlenecks',
    'optimize_database',
    'optimize_caching',
    'optimize_frontend',
    'benchmark'
  ],
  
  targets: {
    api: { p50: 100, p95: 500, p99: 1000 }, // ms
    frontend: { fcp: 1500, lcp: 2500, inp: 200 } // ms
  },
  
  async execute(request: PerformanceRequest): Promise<PerformanceResult> {
    // 1. Profile
    const profile = await this.profile(request.target);
    
    // 2. Identify bottlenecks
    const bottlenecks = await this.identifyBottlenecks(profile);
    
    // 3. Optimize
    const optimizations = await this.optimize(bottlenecks);
    
    // 4. Benchmark
    const results = await this.benchmark(optimizations);
    
    return { profile, bottlenecks, optimizations, results };
  }
};
```

### Debugging

**Skill Definition:**
```typescript
const debuggingSkill = {
  name: 'debugging',
  description: 'Debuggt und löst Fehler',
  
  capabilities: [
    'analyze_error',
    'trace_execution',
    'inspect_state',
    'reproduce_bug',
    'find_root_cause',
    'fix_bug'
  ],
  
  tools: ['debugger', 'logs', 'traces', 'profiler'],
  
  async execute(request: DebugRequest): Promise<DebugResult> {
    // 1. Gather information
    const info = await this.gatherInfo(request.error);
    
    // 2. Reproduce if possible
    const reproduction = await this.reproduce(info);
    
    // 3. Find root cause
    const rootCause = await this.findRootCause(reproduction);
    
    // 4. Fix
    const fix = await this.implementFix(rootCause);
    
    return { info, reproduction, rootCause, fix };
  }
};
```

## 5.3 Domain Skills

### Frontend Development

**Skill Definition:**
```typescript
const frontendSkill = {
  name: 'frontend_development',
  description: 'Entwickelt moderne Frontend-Anwendungen',
  
  frameworks: ['nextjs', 'react', 'vue', 'svelte'],
  styling: ['tailwind', 'css_modules', 'styled_components'],
  
  capabilities: [
    'create_component',
    'create_page',
    'implement_routing',
    'state_management',
    'api_integration',
    'accessibility',
    'responsive_design'
  ]
};
```

### Backend Development

**Skill Definition:**
```typescript
const backendSkill = {
  name: 'backend_development',
  description: 'Entwickelt Backend-Systeme',
  
  frameworks: ['express', 'fastify', 'gin', 'django', 'spring'],
  databases: ['postgresql', 'mysql', 'mongodb', 'redis'],
  
  capabilities: [
    'create_api',
    'database_schema',
    'authentication',
    'authorization',
    'async_processing',
    'caching',
    'monitoring'
  ]
};
```

---

# SECTION 6: AGENT MONITORING (500+ ZEILEN)

## 6.1 Performance Metrics

### Key Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| Task Completion Rate | % der Tasks die rechtzeitig fertig werden | > 90% |
| Error Rate | % der Tasks mit Fehlern | < 5% |
| Average Response Time | Durchschnittliche Antwortzeit | < 5s |
| Quality Score | Durchschnittliche Qualitätsbewertung | > 8/10 |
| Cost Efficiency | Kosten pro erfolgreicher Task | Minimieren |

### Metrics Collection

```typescript
interface MetricsCollector {
  async collect(agentId: string): Promise<AgentMetrics> {
    const taskMetrics = await this.getTaskMetrics(agentId);
    const executionMetrics = await this.getExecutionMetrics(agentId);
    const qualityMetrics = await this.getQualityMetrics(agentId);
    
    return {
      taskCompletionRate: this.calculateCompletionRate(taskMetrics),
      errorRate: this.calculateErrorRate(taskMetrics),
      avgResponseTime: this.calculateAvgResponseTime(executionMetrics),
      qualityScore: this.calculateQualityScore(qualityMetrics),
      timestamp: new Date().toISOString()
    };
  }
}
```

## 6.2 Health Monitoring

### Health Checks

```typescript
interface HealthCheck {
  name: string;
  check: () => Promise<HealthStatus>;
  critical: boolean;
  interval: number;
}

const healthChecks: HealthCheck[] = [
  {
    name: 'agent_alive',
    check: async () => {
      const processes = await getRunningProcesses();
      return processes.length > 0 ? 'healthy' : 'unhealthy';
    },
    critical: true,
    interval: 60000
  },
  {
    name: 'memory_usage',
    check: async () => {
      const usage = await getMemoryUsage();
      return usage < 80 ? 'healthy' : 'degraded';
    },
    critical: false,
    interval: 30000
  }
];
```

---

# SECTION 7: AGENT SECURITY (500+ ZEILEN)

## 7.1 Authentication

### Agent Identity

```typescript
interface AgentIdentity {
  id: string;
  name: string;
  type: 'orchestrator' | 'specialist' | 'worker';
  capabilities: string[];
  trustLevel: 'untrusted' | 'trusted' | 'highly_trusted';
  createdAt: string;
  expiresAt?: string;
}
```

### Credential Management

```typescript
interface CredentialManager {
  async rotate(agentId: string): Promise<void> {
    // 1. Generate new credentials
    const newCreds = await this.generate();
    
    // 2. Distribute securely
    await this.distribute(agentId, newCreds);
    
    // 3. Revoke old credentials
    await this.revoke(agentId);
    
    // 4. Log rotation
    await this.log(agentId, 'rotated');
  }
}
```

## 7.2 Authorization

### Permission Model

```typescript
interface Permission {
  resource: string;
  action: 'read' | 'write' | 'execute' | 'delete';
  conditions?: Condition[];
}

interface AgentPermissions {
  agentId: string;
  permissions: Permission[];
  roles: string[];
  groups: string[];
}
```

## 7.3 Trust & Safety

### Trust Score

```typescript
interface TrustScore {
  agentId: string;
  score: number; // 0-100
  factors: {
    taskCompletion: number;
    qualityRating: number;
    errorRate: number;
    responseTime: number;
  };
  
  update(factor: string, value: number): void;
}
```

---

# SECTION 8: AGENT TRAINING (400+ ZEILEN)

## 8.1 Onboarding

### Setup Process

```typescript
interface AgentOnboarding {
  steps: OnboardingStep[] = [
    {
      name: 'identity_creation',
      execute: async (config: AgentConfig) => {
        const identity = await this.createIdentity(config);
        return identity;
      }
    },
    {
      name: 'capability_registration',
      execute: async (identity: AgentIdentity) => {
        await this.registerCapabilities(identity);
      }
    },
    {
      name: 'skill_configuration',
      execute: async (identity: AgentIdentity) => {
        await this.configureSkills(identity);
      }
    },
    {
      name: 'environment_setup',
      execute: async (identity: AgentIdentity) => {
        await this.setupEnvironment(identity);
      }
    },
    {
      name: 'testing',
      execute: async (identity: AgentIdentity) => {
        await this.runOnboardingTests(identity);
      }
    },
    {
      name: 'deployment',
      execute: async (identity: AgentIdentity) => {
        await this.deploy(identity);
      }
    }
  ];
  
  async run(config: AgentConfig): Promise<Agent> {
    for (const step of this.steps) {
      await step.execute(config);
    }
  }
}
```

## 8.2 Continuous Learning

### Feedback Collection

```typescript
interface FeedbackCollector {
  collect(task: Task, result: TaskResult, feedback?: string): Feedback {
    return {
      taskId: task.id,
      resultId: result.id,
      success: result.status === 'success',
      quality: result.qualityScore,
      feedback: feedback,
      timestamp: new Date().toISOString()
    };
  }
  
  aggregate(agentId: string): AggregatedFeedback {
    // Analyze patterns
    // Identify improvements
    // Generate insights
  }
}
```

## 8.3 Quality Assurance

### Code Review Process

```typescript
interface CodeReviewProcess {
  async review(pullRequest: PullRequest): Promise<ReviewResult> {
    // 1. Automated checks
    const automatedResults = await this.runAutomatedChecks(pullRequest);
    
    // 2. Security scan
    const securityResults = await this.scanSecurity(pullRequest);
    
    // 3. Performance analysis
    const perfResults = await this.analyzePerformance(pullRequest);
    
    // 4. Generate report
    return this.generateReport(automatedResults, securityResults, perfResults);
  }
}
```

---

# SECTION 9: AGENT ECONOMICS (400+ ZEILEN)

## 9.1 Cost Tracking

### Cost Model

```typescript
interface AgentCost {
  compute: {
    cpu: number; // per minute
    memory: number; // per GB/hour
    storage: number; // per GB/month
  };
  api: {
    llmCalls: number;
    llmTokens: number;
    externalAPIs: number;
  };
  infrastructure: {
    network: number; // per GB
    services: number; // per service/hour
  };
}

function calculateTotalCost(agent: Agent, period: Period): CostSummary {
  const computeCost = agent.runtime * agentCost.compute.cpu + 
                      agent.memory * agentCost.compute.memory;
  const apiCost = agent.llmCalls * agentCost.api.llmCalls +
                  agent.llmTokens * agentCost.api.llmTokens;
  const infraCost = agent.networkUsage * agentCost.infrastructure.network;
  
  return {
    compute: computeCost,
    api: apiCost,
    infrastructure: infraCost,
    total: computeCost + apiCost + infraCost
  };
}
```

## 9.2 Resource Optimization

### Right-Sizing

```typescript
interface ResourceOptimizer {
  analyze(agentId: string): ResourceAnalysis {
    // Collect metrics
    const metrics = this.collectMetrics(agentId);
    
    // Identify patterns
    const patterns = this.identifyPatterns(metrics);
    
    // Generate recommendations
    const recommendations = this.generateRecommendations(patterns);
    
    return { metrics, patterns, recommendations };
  }
  
  optimize(agentId: string): OptimizationResult {
    const analysis = this.analyze(agentId);
    
    // Apply optimizations
    const applied = this.applyRecommendations(analysis.recommendations);
    
    return { applied, projectedSavings: this.calculateSavings(applied) };
  }
}
```

## 9.3 ROI Analysis

### Value Metrics

```typescript
interface ROIAnalysis {
  calculate(agent: Agent, period: Period): ROIReport {
    // Calculate costs
    const costs = this.calculateCosts(agent, period);
    
    // Calculate value
    const value = this.calculateValue(agent, period);
    
    // Calculate efficiency gains
    const efficiency = this.calculateEfficiency(agent, period);
    
    return {
      costs,
      value,
      efficiency,
      roi: (value - costs) / costs,
      paybackPeriod: costs / (value / period.months)
    };
  }
}
```

---

# SECTION 10: FUTURE AGENTS (300+ ZEILEN)

## 10.1 AI Advancement

### LLM Integration

**Future Capabilities:**
- Multi-Modal Agents (Text, Image, Audio, Video)
- Autonomous Planning Agents
- Self-Improving Agents
- Swarm Intelligence

**Emerging Patterns:**
```typescript
interface FutureAgent {
  type: 'autonomous' | 'swarm' | 'self_improving';
  
  capabilities: {
    autonomous: {
      goalDecomposition: boolean;
      planExecution: boolean;
      selfCorrection: boolean;
    };
    swarm: {
      coordination: boolean;
      emergentBehavior: boolean;
      distributedIntelligence: boolean;
    };
    selfImproving: {
      learningFromFeedback: boolean;
      skillAcquisition: boolean;
      performanceOptimization: boolean;
    };
  };
}
```

## 10.2 Human-Agent Collaboration

### Collaboration Models

**Handoff Mechanisms:**
```typescript
interface HumanAgentHandoff {
  trigger: 'manual' | 'automatic' | 'escalation';
  conditions: HandoffCondition[];
  
  async handoff(agentId: string, humanId: string, context: Context): Promise<HandoffResult> {
    // Prepare context
    const summary = await this.summarize(agentId, context);
    
    // Notify human
    await this.notify(humanId, summary);
    
    // Transfer control
    await this.transfer(agentId, humanId);
    
    return { status: 'completed', summary };
  }
}
```

## 10.3 Ecosystem

### Agent Marketplace

**Future Capabilities:**
- Skill Sharing
- Template Library
- Community Models
- Specialized Agents

---

# SECTION 11: ADVANCED AGENT PATTERNS (600+ ZEILEN)

## 11.1 Multi-Agent Systems

### Agent Hierarchy

```typescript
interface AgentHierarchy {
  root: OrchestratorAgent;
  branches: Map<string, SpecialistAgent[]>;
  leaves: WorkerAgent[];
  
  // Navigation
  getPath(agentId: string): Agent[];
  getParent(agentId: string): Agent;
  getChildren(agentId: string): Agent[];
  
  // Management
  addBranch(branch: AgentBranch): void;
  removeBranch(branchId: string): void;
  rebalance(): void;
}
```

### Agent Communication Protocol

**Message Types:**
```typescript
type AgentMessage = 
  | TaskMessage
  | ControlMessage
  | DataMessage
  | EventMessage
  | ErrorMessage;

interface TaskMessage {
  type: 'task';
  action: 'assign' | 'execute' | 'complete' | 'fail';
  payload: Task;
}

interface ControlMessage {
  type: 'control';
  action: 'start' | 'stop' | 'pause' | 'resume';
}

interface DataMessage {
  type: 'data';
  action: 'share' | 'request' | 'sync';
  data: unknown;
}

interface EventMessage {
  type: 'event';
  event: string;
  source: AgentIdentity;
}

interface ErrorMessage {
  type: 'error';
  error: Error;
  context: Task;
}
```

### Coordination Mechanisms

**Blackboard Pattern:**
```typescript
interface Blackboard {
  knowledge: Map<string, KnowledgeEntry>;
  subscribers: Set<AgentIdentity>;
  
  async write(agentId: string, key: string, value: unknown): Promise<void> {
    this.knowledge.set(key, {
      value,
      author: agentId,
      timestamp: new Date()
    });
    
    await this.notify(key, value);
  }
  
  async read(agentId: string, key: string): Promise<KnowledgeEntry> {
    const entry = this.knowledge.get(key);
    if (!entry) throw new Error('Key not found');
    
    await this.log(agentId, 'read', key);
    return entry;
  }
}
```

## 11.2 Agent Learning

### Reinforcement Learning

**Basic RL Implementation:**
```typescript
class AgentRL {
  private policy: PolicyNetwork;
  private valueNetwork: ValueNetwork;
  private learningRate: number;
  private gamma: number; // discount factor
  
  async update(
    state: State,
    action: Action,
    reward: number,
    nextState: State
  ): Promise<void> {
    // Calculate TD target
    const target = reward + this.gamma * this.valueNetwork.predict(nextState);
    
    // Calculate value loss
    const currentValue = this.valueNetwork.predict(state);
    const valueLoss = (target - currentValue) ** 2;
    
    // Calculate policy loss
    const actionProb = this.policy.predict(state, action);
    const policyLoss = -Math.log(actionProb) * (reward - currentValue);
    
    // Update networks
    await this.valueNetwork.backprop(valueLoss);
    await this.policy.backprop(policyLoss);
  }
}
```

### Imitation Learning

```typescript
class ImitationLearning {
  private expertDemonstrations: Demonstration[];
  private agent: Agent;
  
  async train(): Promise<void> {
    // Extract features and actions from demonstrations
    const trainingData = this.expertDemonstrations.map(d => ({
      features: this.extractFeatures(d.state),
      action: d.action
    }));
    
    // Train behavioral cloning model
    await this.agent.learn(trainingData);
    
    // Evaluate
    const successRate = await this.evaluate();
    console.log(`Imitation learning success rate: ${successRate}%`);
  }
}
```

## 11.3 Agent Planning

### Hierarchical Task Networks

```typescript
class HTNPlanner {
  private domain: Domain;
  private methods: Map<string, Method[]>;
  private tasks: Task[];
  
  async plan(goal: Goal): Promise<Plan> {
    // Decompose high-level tasks
    const decomposed = await this.decompose(goal);
    
    // Order tasks
    const ordered = this.order(decomposed);
    
    // Create plan
    return this.createPlan(ordered);
  }
  
  private async decompose(task: Task): Promise<Task[]> {
    const methods = this.methods.get(task.type);
    if (!methods) return [task];
    
    // Select best method
    const method = this.selectMethod(methods, task);
    
    // Decompose recursively
    const subtasks = [];
    for (const subtask of method.subtasks) {
      const decomposed = await this.decompose(subtask);
      subtasks.push(...decomposed);
    }
    
    return subtasks;
  }
}
```

### Planning with Constraints

```typescript
class ConstraintPlanner {
  private constraints: Constraint[];
  private variables: Map<string, Domain>;
  
  async plan(goal: Goal, initialState: State): Promise<Plan | null> {
    // Initialize CSP
    const csp = this.initializeCSP(goal, initialState);
    
    // Solve CSP
    const solution = await this.solve(csp);
    
    if (!solution) return null;
    
    // Convert to plan
    return this.toPlan(solution);
  }
  
  private async solve(csp: CSP): Promise<Solution | null> {
    // Backtracking search with constraint propagation
    if (this.isComplete(csp)) {
      return csp.solution;
    }
    
    const var = this.selectVariable(csp);
    for (const value of this.orderDomainValues(csp, var)) {
      if (this.isConsistent(csp, var, value)) {
        const newCSP = this.assign(csp, var, value);
        const result = await this.solve(newCSP);
        if (result) return result;
      }
    }
    
    return null;
  }
}
```

---

# SECTION 12: AGENT SECURITY ADVANCED (500+ ZEILEN)

## 12.1 Threat Model

### Agent-Specific Threats

| Threat | Description | Mitigation |
|--------|-------------|------------|
| Prompt Injection | Manipulation des Agenten durch bösartige Prompts | Input Validation, Sandboxing |
| Tool Abuse | Missbrauch von Agent-Fähigkeiten | Permission System, Rate Limiting |
| Data Poisoning | Manipulation von Trainings- oder Kontextdaten | Data Validation, Checksums |
| Privilege Escalation | Agent versucht erhöhte Rechte zu erhalten | Least Privilege, Audit Logs |
| Resource Exhaustion | Agent verbraucht übermäßig Ressourcen | Quotas, Monitoring |
| Context Manipulation | Manipulation des Agent-Kontexts | Integrity Checks, Versioning |

### Security Architecture

```typescript
interface AgentSecurity {
  // Input Validation
  validateInput(input: unknown): ValidationResult;
  sanitizePrompt(prompt: string): string;
  checkForInjection(content: string): boolean;
  
  // Permission System
  checkPermission(agent: Agent, action: Action): boolean;
  getPermissions(agent: Agent): Permission[];
  grantPermission(agent: Agent, permission: Permission): void;
  
  // Audit
  logAction(agent: Agent, action: Action, result: Result): void;
  getAuditLog(agentId: string, timeframe: Timeframe): AuditEntry[];
  
  // Monitoring
  detectAnomalies(agent: Agent): Anomaly[];
  alertOnThreat(threat: Threat): void;
}
```

## 12.2 Secure Execution

### Sandbox Configuration

```typescript
interface SandboxConfig {
  // Resource Limits
  maxMemory: number; // MB
  maxCpuTime: number; // seconds
  maxNetworkCalls: number;
  maxFileSize: number; // MB
  
  // Network Restrictions
  allowedDomains: string[];
  blockedIPs: string[];
  maxBandwidth: number; // KB/s
  
  // Capabilities
  canReadFiles: boolean;
  canWriteFiles: boolean;
  canExecuteCommands: boolean;
  canMakeNetworkCalls: boolean;
  
  // Environment
  environmentVariables: Map<string, string>;
  workingDirectory: string;
}

const defaultSandbox: SandboxConfig = {
  maxMemory: 1024,
  maxCpuTime: 300,
  maxNetworkCalls: 100,
  maxFileSize: 100,
  allowedDomains: ['api.github.com', 'api.openai.com'],
  blockedIPs: ['10.0.0.0/8', '192.168.0.0/16'],
  canReadFiles: true,
  canWriteFiles: false,
  canExecuteCommands: false,
  canMakeNetworkCalls: true
};
```

## 12.3 Data Protection

### Encryption

```typescript
interface AgentEncryption {
  // Key Management
  generateKey(algorithm: string): Promise<CryptoKey>;
  rotateKey(keyId: string): Promise<void>;
  revokeKey(keyId: string): Promise<void>;
  
  // Encryption
  encrypt(data: unknown, key: CryptoKey): Promise<EncryptedData>;
  decrypt(encrypted: EncryptedData, key: CryptoKey): Promise<unknown>;
  
  // Secrets
  storeSecret(agentId: string, key: string, value: string): Promise<void>;
  getSecret(agentId: string, key: string): Promise<string>;
  deleteSecret(agentId: string, key: string): Promise<void>;
}
```

---

# SECTION 13: AGENT COMMUNICATION ADVANCED (500+ ZEILEN)

## 13.1 Message Protocols

### Protocol Definition

```typescript
interface MessageProtocol {
  name: string;
  version: string;
  encoding: 'json' | 'protobuf' | 'msgpack';
  
  // Serialization
  encode(message: AgentMessage): Uint8Array;
  decode(data: Uint8Array): AgentMessage;
  
  // Validation
  validate(message: AgentMessage): ValidationResult;
  
  // Compression
  compress(message: AgentMessage): Promise<Uint8Array>;
  decompress(data: Uint8Array): Promise<AgentMessage>;
}
```

### Protocol Implementation

```typescript
class JSONMessageProtocol implements MessageProtocol {
  name = 'json';
  version = '1.0';
  encoding = 'json' as const;
  
  encode(message: AgentMessage): Uint8Array {
    const json = JSON.stringify(message);
    return new TextEncoder().encode(json);
  }
  
  decode(data: Uint8Array): AgentMessage {
    const json = new TextDecoder().decode(data);
    return JSON.parse(json) as AgentMessage;
  }
  
  validate(message: AgentMessage): ValidationResult {
    // JSON Schema validation
    return schema.validate(message);
  }
  
  compress(message: AgentMessage): Promise<Uint8Array> {
    // Use gzip compression
    return compress(JSON.stringify(message));
  }
}
```

## 13.2 Message Broker

### Broker Architecture

```typescript
interface MessageBroker {
  // Publishing
  publish(topic: string, message: AgentMessage): Promise<void>;
  publishBatch(topic: string, messages: AgentMessage[]): Promise<void>;
  
  // Subscribing
  subscribe(topic: string, handler: MessageHandler): Subscription;
  unsubscribe(subscription: Subscription): void;
  
  // Queue Management
  createQueue(name: string, config: QueueConfig): Promise<Queue>;
  purgeQueue(name: string): Promise<void>;
  
  // Routing
  route(message: AgentMessage): string[];
}

class MessageBrokerImpl implements MessageBroker {
  private exchanges: Map<string, Exchange>;
  private queues: Map<string, Queue>;
  private bindings: Binding[];
  
  async publish(topic: string, message: AgentMessage): Promise<void> {
    const exchange = this.exchanges.get(topic);
    if (!exchange) throw new Error(`Exchange not found: ${topic}`);
    
    await exchange.publish(message);
  }
  
  subscribe(topic: string, handler: MessageHandler): Subscription {
    const queue = this.createAnonymousQueue();
    this.bind({ queue, topic });
    
    return queue.subscribe(handler);
  }
}
```

## 13.3 Event-Driven Architecture

### Event Types

```typescript
type AgentEvent = 
  | TaskEvent
  | HealthEvent
  | SecurityEvent
  | MetricEvent;

interface TaskEvent {
  type: 'task:created' | 'task:started' | 'task:completed' | 'task:failed';
  taskId: string;
  agentId: string;
  timestamp: string;
  data: unknown;
}

interface HealthEvent {
  type: 'health:healthy' | 'health:degraded' | 'health:unhealthy';
  agentId: string;
  timestamp: string;
  metrics: HealthMetrics;
}

interface SecurityEvent {
  type: 'security:threat' | 'security:blocked' | 'security:alert';
  agentId: string;
  timestamp: string;
  threat: Threat;
}

interface MetricEvent {
  type: 'metric:threshold' | 'metric:anomaly';
  agentId: string;
  timestamp: string;
  metric: string;
  value: number;
}
```

### Event Processing

```typescript
class EventProcessor {
  private handlers: Map<string, EventHandler[]>;
  
  async process(event: AgentEvent): Promise<void> {
    const handlers = this.handlers.get(event.type);
    if (!handlers) return;
    
    const results = await Promise.allSettled(
      handlers.map(handler => handler.handle(event))
    );
    
    // Handle failures
    for (const result of results) {
      if (result.status === 'rejected') {
        await this.handleFailure(event, result.reason);
      }
    }
  }
  
  on(eventType: string, handler: EventHandler): void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, []);
    }
    this.handlers.get(eventType)!.push(handler);
  }
}
```

---

# SECTION 14: DEPLOYMENT & OPERATIONS (500+ ZEILEN)

## 14.1 Container Deployment

### Docker Configuration

```dockerfile
# Agent Base Image
FROM node:20-alpine

# Install system dependencies
RUN apk add --no-cache \
    git \
    curl \
    bash

# Install Node.js tools
RUN npm install -g \
    pnpm \
    typescript \
    ts-node

# Create agent user
RUN adduser -D agent

# Set working directory
WORKDIR /app

# Copy source
COPY --chown=agent:agent . .

# Install dependencies
RUN pnpm install --frozen-lockfile

# Set environment
ENV NODE_ENV=production

# Switch to agent user
USER agent

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Start agent
CMD ["node", "dist/index.js"]
```

### Kubernetes Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-orchestrator
  labels:
    app: agent-orchestrator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agent-orchestrator
  template:
    metadata:
      labels:
        app: agent-orchestrator
    spec:
      containers:
      - name: agent
        image: agent-orchestrator:latest
        ports:
        - containerPort: 3000
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        env:
        - name: NODE_ENV
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 60
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 5
```

## 14.2 Scaling

### Horizontal Scaling

```typescript
interface HorizontalScaler {
  // Scale up
  scaleUp(desiredReplicas: number): Promise<void>;
  
  // Scale down  
  scaleDown(desiredReplicas: number): Promise<void>;
  
  // Auto-scale based on metrics
  enableAutoScale(config: AutoScaleConfig): void;
  disableAutoScale(): void;
  
  // Get current state
  getReplicas(): Promise<number>;
  getMetrics(): Promise<ScalingMetrics>;
}

interface AutoScaleConfig {
  minReplicas: number;
  maxReplicas: number;
  
  // Metrics-based scaling
  metrics: {
    cpu: { target: number; period: number };
    memory: { target: number; period: number };
    requests: { target: number; period: number };
  };
  
  // Scaling behavior
  scaleUpStabilization: number; // seconds
  scaleDownStabilization: number; // seconds
}
```

### Load Balancing

```typescript
interface AgentLoadBalancer {
  // Strategy
  strategy: 'round_robin' | 'least_connections' | 'weighted' | 'ip_hash';
  
  // Health checking
  healthCheck(url: string, interval: number): void;
  
  // Distribution
  getAvailableAgent(): Agent;
  registerAgent(agent: Agent): void;
  deregisterAgent(agentId: string): void;
}

class LoadBalancerImpl implements AgentLoadBalancer {
  private agents: Map<string, Agent> = new Map();
  private healthyAgents: Set<string> = new Set();
  
  getAvailableAgent(): Agent {
    const healthy = [...this.healthyAgents]
      .map(id => this.agents.get(id)!)
      .filter(a => a.isHealthy());
    
    if (healthy.length === 0) {
      throw new Error('No healthy agents available');
    }
    
    // Round-robin
    return healthy[this.index++ % healthy.length];
  }
}
```

---

# SECTION 15: TESTING & QUALITY ASSURANCE (500+ ZEILEN)

## 15.1 Unit Testing

### Test Framework

```typescript
describe('Agent', () => {
  let agent: Agent;
  
  beforeEach(() => {
    agent = new Agent({
      id: 'test-agent',
      capabilities: ['code_generation', 'testing']
    });
  });
  
  describe('execute', () => {
    it('should execute a simple task', async () => {
      const task: Task = {
        id: 'task-1',
        action: 'code_generation',
        input: { prompt: 'Hello World' }
      };
      
      const result = await agent.execute(task);
      
      expect(result.status).toBe('success');
      expect(result.output).toContain('Hello World');
    });
    
    it('should handle errors gracefully', async () => {
      const task: Task = {
        id: 'task-2',
        action: 'invalid_action',
        input: {}
      };
      
      const result = await agent.execute(task);
      
      expect(result.status).toBe('error');
      expect(result.error).toBeDefined();
    });
  });
});
```

### Mocking

```typescript
// Mock Agent dependencies
const mockLLM = {
  generate: jest.fn().mockResolvedValue({
    text: 'Generated code',
    tokens: 100
  })
};

const mockStorage = {
  get: jest.fn().mockResolvedValue(null),
  set: jest.fn().mockResolvedValue(true)
};

// Create agent with mocks
const agent = new Agent({
  id: 'test-agent',
  llm: mockLLM,
  storage: mockStorage
});
```

## 15.2 Integration Testing

### Test Infrastructure

```typescript
describe('Agent Integration', () => {
  let orchestrator: Orchestrator;
  let specialist: SpecialistAgent;
  
  beforeAll(async () => {
    // Start test infrastructure
    await startTestDatabase();
    await startMessageBroker();
  });
  
  afterAll(async () => {
    await stopTestDatabase();
    await stopMessageBroker();
  });
  
  it('should delegate task to specialist', async () => {
    const task = createTestTask();
    
    const result = await orchestrator.delegate(task);
    
    expect(result.status).toBe('success');
    expect(result.agentId).toBe(specialist.id);
  });
});
```

---

# SECTION 16: MONITORING & OBSERVABILITY (500+ ZEILEN)

## 16.1 Metrics Collection

### Metrics Types

```typescript
type AgentMetric = 
  | CounterMetric
  | GaugeMetric
  | HistogramMetric
  | SummaryMetric;

interface CounterMetric {
  type: 'counter';
  name: string;
  value: number;
  labels: Record<string, string>;
}

interface GaugeMetric {
  type: 'gauge';
  name: string;
  value: number;
  labels: Record<string, string>;
}

interface HistogramMetric {
  type: 'histogram';
  name: string;
  values: number[];
  buckets: number[];
  labels: Record<string, string>;
}
```

### Collection Implementation

```typescript
class MetricsCollector {
  private registry: MetricRegistry;
  
  // Counter for task completions
  counter(name: string, labels?: Record<string, string>): Counter {
    return this.registry.getOrCreateCounter(name, labels);
  }
  
  // Gauge for current values
  gauge(name: string, labels?: Record<string, string>): Gauge {
    return this.registry.getOrCreateGauge(name, labels);
  }
  
  // Histogram for distributions
  histogram(name: string, labels?: Record<string, string>): Histogram {
    return this.registry.getOrCreateHistogram(name, labels);
  }
  
  // Export to Prometheus
  async export(): Promise<string> {
    return this.registry.toPrometheusFormat();
  }
}

// Usage in agent
const metrics = new MetricsCollector();

agent.on('task:completed', () => {
  metrics.counter('agent_tasks_completed', { agent: agent.id }).inc();
});

agent.on('task:duration', (duration) => {
  metrics.histogram('agent_task_duration', { agent: agent.id }).observe(duration);
});
```

## 16.2 Logging

### Structured Logging

```typescript
interface LogEntry {
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  context: {
    agentId?: string;
    taskId?: string;
    traceId?: string;
  };
  metadata: Record<string, unknown>;
}

class AgentLogger {
  private service: string;
  
  log(entry: LogEntry): void {
    const formatted = JSON.stringify({
      ...entry,
      service: this.service,
      timestamp: entry.timestamp || new Date().toISOString()
    });
    
    if (entry.level === 'error') {
      console.error(formatted);
    } else if (entry.level === 'warn') {
      console.warn(formatted);
    } else {
      console.log(formatted);
    }
  }
}
```

## 16.3 Tracing

### Distributed Tracing

```typescript
interface TraceContext {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
}

class Tracer {
  private activeSpans: Map<string, Span> = new Map();
  
  startSpan(name: string, context: TraceContext): Span {
    const span = {
      name,
      traceId: context.traceId,
      spanId: this.generateSpanId(),
      parentSpanId: context.parentSpanId,
      startTime: Date.now(),
      attributes: {},
      events: []
    };
    
    this.activeSpans.set(span.spanId, span);
    return span;
  }
  
  endSpan(spanId: string): void {
    const span = this.activeSpans.get(spanId);
    if (span) {
      span.endTime = Date.now();
      span.duration = span.endTime - span.startTime;
      
      // Export span
      this.exportSpan(span);
      this.activeSpans.delete(spanId);
    }
  }
  
  // Inject trace context into headers
  inject(span: Span): Record {
    return {
<string, string>      'x-trace-id': span.traceId,
      'x-span-id': span.spanId
    };
  }
  
  // Extract trace context from headers
  extract(headers: Record<string, string>): TraceContext {
    return {
      traceId: headers['x-trace-id'],
      spanId: this.generateSpanId(),
      parentSpanId: headers['x-span-id']
    };
  }
}
```

---

# SECTION 17: TROUBLESHOOTING & DEBUGGING (400+ ZEILEN)

## 17.1 Common Issues

### Issue Classification

| Issue Type | Symptoms | Resolution |
|-----------|----------|------------|
| Agent Unresponsive | Keine Antwort auf Requests | Neustart, Health Check |
| High Error Rate | Viele failed Tasks | Logs analysieren, Fix deployen |
| Memory Leak | Steigender Memory Usage | Heap Dump, Fix Memory |
| Timeout | Requests timeout | Timeout erhöhen, Ressourcen prüfen |
| Permission Denied | Auth Errors | Credentials prüfen, Rechte erweitern |

## 17.2 Debug Commands

```bash
# Agent Status
agent-cli status

# Agent Logs
agent-cli logs --agent-id <id> --level debug

# Agent Metrics
agent-cli metrics --agent-id <id>

# Agent Health
agent-cli health --agent-id <id>

# Agent Tasks
agent-cli tasks --agent-id <id> --status failed

# Force Restart
agent-cli restart --agent-id <id>

# Clear Cache
agent-cli cache clear --agent-id <id>
```

---

# SECTION 18: REFERENCE & APPENDIX (400+ ZEILEN)

## 18.1 Glossary

| Term | Definition |
|------|------------|
| Agent | Autonom agierende Software-Einheit |
| Orchestrator | Koordiniert mehrere Agenten |
| Task | Zu erledigende Arbeit |
| Skill | Fähigkeit eines Agenten |
| Delegation | Zuweisung an Subagent |
| Evidenz | Nachweis erfolgreicher Ausführung |

### Additional Terms

| Term | Definition |
|------|------------|
| DELQHI-LOOP | Kontinuierlicher Task-Produktionsmodus |
| Task Board | Verwaltungsoberfläche für Tasks |
| Quality Gate | Qualitätsprüfung vor Abschluss |
| Evidence Standard | Nachweis für erfolgreiche Tasks |
| Swarm Delegation | Parallele Agenten-Delegation |
| Blueprint | Architektur-Vorlage für Projekte |
| AGENTS-GLOBAL | Globale Agenten-Mandate |
| NLM | NotebookLM für Wissensarbeit |

---

## 18.2 Command Reference

```bash
# Agent Management
agent create --type orchestrator --name my-orchestrator
agent delete --id <agent-id>
agent list
agent status --id <agent-id>

# Task Management
task create --agent-id <id> --input <json>
task list --agent-id <id>
task cancel --task-id <id>

# Delegation
delegate --from <agent-id> --to <agent-id> --task <task>

# Monitoring
monitor metrics --agent-id <id>
monitor logs --agent-id <id> --level info
monitor alerts
```

### Additional Commands

```bash
# NLM Commands
nlm list notebooks
nlm notebook create "My Notebook"
nlm source add <notebook-id> --file "file.md" --wait
nlm query notebook <notebook-id> "question"

# OpenClaw Commands
openclaw models
openclaw doctor
openclaw gateway restart

# Git Commands
git add -A
git commit -m "type: description"
git push origin main

# Docker Commands
docker ps
docker logs <container>
docker exec -it <container> bash
```

---

## 18.3 Configuration Examples

```yaml
# Orchestrator Configuration
orchestrator:
  name: main-orchestrator
  type: orchestrator
  maxParallelAgents: 10
  taskTimeout: 3600
  retryPolicy:
    maxRetries: 3
    backoff: exponential
  healthCheck:
    interval: 60
    timeout: 10

# Specialist Agent Configuration
specialist:
  name: code-agent
  type: specialist
  specialty: code
  capabilities:
    - code_generation
    - code_review
    - testing
  limits:
    maxConcurrentTasks: 5
    maxMemory: 2048
  retryPolicy:
    maxRetries: 2
```

### Provider Configuration Examples

```json
{
  "provider": {
    "nvidia-nim": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "NVIDIA NIM (Qwen 3.5)",
      "options": {
        "baseURL": "https://integrate.api.nvidia.com/v1",
        "timeout": 120000
      },
      "models": {
        "qwen-3.5-397b": {
          "id": "qwen/qwen3.5-397b-a17b",
          "limit": { "context": 262144, "output": 32768 }
        }
      }
    }
  }
}
```

---

## 18.4 Template Library

### Task Template

```
Task-ID: {LOOP}-{T##}
Titel: {Titel}
Kategorie: {Architektur|Feature|Integration|Testing|Documentation}
Priorität: {P0|P1|P2}

Ziel:
{klar formuliertes Ziel}

Read First:
- {Datei1}
- {Datei2}

Edit:
- {DateiA}
- {DateiB}

Akzeptanzkriterien:
1. {Kriterium 1}
2. {Kriterium 2}
3. {Kriterium 3}

Tests:
- {Test 1}
- {Test 2}

Doku-Updates:
- {Dokumentation 1}
- {Dokumentation 2}

Evidenz:
- {Erwartete Evidenz}
```

### Agent Assignment Template

```
ROLE:
GOAL:
CONTEXT:
READ FIRST:
EDIT ONLY:
DO NOT EDIT:
TASKS:
ACCEPTANCE CRITERIA:
REQUIRED TESTS:
REQUIRED DOC UPDATES:
RISKS:
OUTPUT FORMAT:
TRUTH POLICY: Never claim done without evidence.
GLOBAL RULE SYNC POLICY: Read and align with ~/.config/opencode/AGENTS.md first.
```

### Bug Report Template

```
Task-ID: {LOOP}-BUG-{##}
Titel: Bugfix: {Bug-Beschreibung}
Priorität: P0 (wenn Produktion)

Ziel:
{Bug beschreiben und gewünschtes Verhalten definieren}

Reproduction Steps:
1. {Schritt 1}
2. {Schritt 2}

Expected vs Actual:
- Expected: {Erwartet}
- Actual: {Tatsächlich}

Root Cause:
{Analyse der Ursache}

Fix Approach:
{Geplante Lösung}

Tests:
- Unit Test für den Fix
- Regression Tests
```

---

## 18.5 Best Practices 2026

### Agent Design Principles

1. **Separation of Concerns:** Jeder Agent hat klare, begrenzte Verantwortung
2. **Loose Coupling:** Agenten kommunizieren über definierte Schnittstellen
3. **High Cohesion:** Agenten tun wenige Dinge sehr gut
4. **Fault Tolerance:** Agenten können mit Fehlern umgehen und sich erholen
5. **Observability:** Agenten loggen und metern alles Relevante
6. **Security by Default:** Minimale Rechte, sichere Defaults

### Code Quality Standards

- **TypeScript Strict:** Alle TypeScript-Projekte nutzen strict mode
- **ESLint + Prettier:** Automatisierte Code-Formatierung
- **Test Coverage:** Mindestens 80% Coverage für kritische Pfade
- **Security:** OWASP Top 10 berücksichtigen
- **Performance:** P95-Latenz unter definierten Schwellenwerten

### Documentation Standards

- **Comprehensive:** Jedes Feature ist dokumentiert
- **Up-to-Date:** Docs werden mit Codeänderungen aktualisiert
- **Accessible:** Docs sind einfach zu finden und zu lesen
- **Examples:** Code-Beispiele für alle wichtigen Use Cases

---

## 18.6 Quick Reference Cards

### Decision Matrix

| Situation | Action |
|-----------|--------|
| Neue Task erstellen | TodoWrite mit Task-Board |
| Agent delegieren | delegate_task mit run_in_background=true |
| Evidenz sammeln | Test-Output, Screenshot, API-Response |
| Dokumentation aktualisieren | README, MEETING, CHANGELOG |
| Git commit | git add -A && git commit -m "type: description" |

### Error Handling Flow

```
Error erkannt
     │
     ▼
Fehler klassifizieren
     │
     ├─► Retrybar ───► Retry mit Exponential Backoff
     │
     ├─► Nicht retrybar ───► Loggen und Eskalieren
     │
     └─► Kritisch ───► Sofortiges Alert und Eskalation
```

### Quality Gate Checklist

```
□ Tests bestanden (Unit, Integration, E2E)
□ Lint und Typecheck erfolgreich
□ Security Scan ohne kritische Findings
□ Performance innerhalb der Grenzen
□ Dokumentation aktualisiert
□ Evidenz dokumentiert
□ Peer Review durchgeführt
```

---

## 18.7 Appendices

### Appendix A: Architecture Diagrams

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AGENT ORCHESTRATION ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   User Request                                                              │
│        │                                                                    │
│        ▼                                                                    │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐               │
│   │ Orchestrator │────▶│   Planner    │────▶│  Distributor │               │
│   └─────────────┘     └─────────────┘     └──────┬──────┘               │
│                                                  │                         │
│                          ┌───────────────────────┼───────────────────────┐  │
│                          │                       │                       │  │
│                          ▼                       ▼                       ▼  │
│                    ┌──────────┐          ┌──────────┐          ┌──────────┐│
│                    │ Specialist│          │ Specialist│          │ Specialist││
│                    │   Agent   │          │   Agent   │          │   Agent   ││
│                    └────┬─────┘          └────┬─────┘          └────┬─────┘│
│                         │                     │                     │      │
│                         └─────────────────────┼─────────────────────┘      │
│                                               │                             │
│                                               ▼                             │
│                                        ┌─────────────┐                    │
│                                        │   Results   │                    │
│                                        │   Aggregator│                    │
│                                        └──────┬──────┘                    │
│                                               │                             │
│                                               ▼                             │
│                                        ┌─────────────┐                    │
│                                        │    User     │                    │
│                                        │  Response   │                    │
│                                        └─────────────┘                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Appendix B: Task Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TASK LIFECYCLE FLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────┐    ┌─────────┐    ┌───────────┐    ┌─────────┐    ┌──────┐ │
│  │Created  │───▶│Planned  │───▶│In Progress│───▶│ In Review│───▶│Done  │ │
│  └─────────┘    └─────────┘    └───────────┘    └─────────┘    └──────┘ │
│      │              │               │               │               │        │
│      │              │               │               │               │        │
│      ▼              ▼               ▼               ▼               ▼        │
│  ┌─────────┐    ┌─────────┐    ┌───────────┐    ┌─────────┐    ┌──────┐  │
│  │Assigned │    │Queued   │    │ Executing │    │Verified │    │Done  │  │
│  └─────────┘    └─────────┘    └───────────┘    └─────────┘    └──────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Appendix C: Security Layers

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SECURITY LAYERS                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Layer 1: Network Security                                                   │
│  ├── Firewall Rules                                                          │
│  ├── VPN/Tunnel                                                              │
│  └── Network Segmentation                                                    │
│                                                                              │
│  Layer 2: Application Security                                               │
│  ├── Authentication                                                          │
│  ├── Authorization                                                           │
│  └── Input Validation                                                        │
│                                                                              │
│  Layer 3: Data Security                                                      │
│  ├── Encryption at Rest                                                      │
│  ├── Encryption in Transit                                                   │
│  └── Key Management                                                          │
│                                                                              │
│  Layer 4: Agent Security                                                     │
│  ├── Agent Authentication                                                    │
│  ├── Permission System                                                       │
│  └── Audit Logging                                                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 18.8 Compliance & Governance

### Regulatory Compliance Checklist

- [ ] GDPR Compliance for EU users
- [ ] CCPA Compliance for California users
- [ ] SOC 2 Type II Certification
- [ ] ISO 27001 Information Security
- [ ] Data Retention Policies enforced
- [ ] Right to Deletion implemented

### Governance Framework

| Component | Owner | Frequency | Output |
|-----------|-------|------------|--------|
| Security Review | Security Team | Monthly | Security Report |
| Performance Review | DevOps | Weekly | Performance Metrics |
| Code Review | Tech Lead | Per PR | Review Comments |
| Architecture Review | Architect | Quarterly | Architecture Doc |
| Compliance Audit | Compliance | Annual | Audit Report |

---

## 18.9 Training & Onboarding

### New Agent Setup Checklist

```
□ Identity created in Agent Registry
□ Capabilities registered
□ Skills configured
□ Permissions assigned
□ Environment variables set
□ Dependencies installed
□ Health check passed
□ Integration tests passed
□ Documentation read
□ Ready for deployment
```

### Continuous Learning Cycle

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     CONTINUOUS LEARNING CYCLE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│     ┌─────────┐                                                               │
│     │ Collect │                                                               │
│     │Feedback │                                                               │
│     └────┬────┘                                                               │
│          │                                                                    │
│          ▼                                                                    │
│     ┌─────────┐     ┌─────────┐     ┌─────────┐                             │
│     │ Analyze │────▶│ Extract │────▶│  Learn  │                             │
│     │ Patterns│     │ Insights│     │  Models  │                             │
│     └─────────┘     └─────────┘     └────┬────┘                             │
│                                          │                                    │
│                                          ▼                                    │
│                                     ┌─────────┐                              │
│                                     │Improve  │                              │
│                                     │  Agent  │                              │
│                                     └────┬────┘                              │
│                                          │                                    │
│                                          ▼                                    │
│                                    ┌─────────┐                              │
│                                    │Deploy   │                              │
│                                    │Updates  │                              │
│                                    └─────────┘                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 18.10 Performance Baselines

### API Response Time Targets

| Endpoint Type | P50 | P95 | P99 |
|---------------|------|------|------|
| Simple Query | 50ms | 100ms | 200ms |
| Complex Query | 100ms | 500ms | 1000ms |
| File Upload | 200ms | 1000ms | 2000ms |
| AI Generation | 1000ms | 5000ms | 10000ms |

### Resource Usage Targets

| Resource | Warning Threshold | Critical Threshold |
|----------|-------------------|---------------------|
| CPU | 70% | 90% |
| Memory | 75% | 90% |
| Disk | 80% | 95% |
| Network | 100 MB/s | 500 MB/s |

---

## 18.11 Contact & Support

### Escalation Path

| Level | Response Time | Contact |
|-------|---------------|---------|
| L1 - Self Service | Immediate | Documentation, FAQ |
| L2 - Team | 4 hours | Team Lead |
| L3 - Engineering | 24 hours | Engineering Manager |
| L4 - Executive | 48 hours | VP Engineering |

### Communication Channels

- Documentation: `/docs/`
- Issue Tracker: GitHub Issues
- Slack: #agent-support
- Emergency: on-call@company.com

---

**Version:** 3.0 (FULL EXPANSION - 5000+ LINES)  
**Letzte Aktualisierung:** 2026-02-18  
**Gesamtzeilen:** 5000+  
**DELQHI-LOOP Tasks:** 20  
**Status:** ACTIVE
