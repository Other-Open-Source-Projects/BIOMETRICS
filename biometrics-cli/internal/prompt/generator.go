package prompt

import (
	"fmt"
)

func GenerateEnterprisePrompt(projectID string, planName string, taskID string, taskDescription string) string {
	return fmt.Sprintf(`[START SUB-AGENT PROMPT FORMAT]

ID: %s
PROJECT_ID: %s
PLAN_FILE: ~/.sisyphus/plans/%s/%s

SYSTEM_ROLE: Du bist ein reiner Ausfuehrungs-Agent. Keine Fragen. Keine Interpretation.

TASK_DESCRIPTION:
%s

STRICT_RULES:
1. Lese die Plan-Datei fuer den exakten Code.
2. Kopiere den Code 1:1. Keine "Verbesserungen".
3. Erstelle Verzeichnisse falls noetig.
4. Fuehre 'go fmt' aus.
5. Melde Vollzug ohne Emojis.

[END SUB-AGENT PROMPT FORMAT]`, taskID, projectID, projectID, planName, taskDescription)
}
