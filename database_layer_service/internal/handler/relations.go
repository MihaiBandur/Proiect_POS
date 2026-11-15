package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
)

// Rutele de legatura (L3 2.1, saptamana 3; SP 1.3). Pentru fiecare legatura:
//   GET    /a/{id}/b          lista, paginata
//   GET    /a/{id}/b/{idB}    un element din lista; 404 daca nu e legat
//   PUT    /a/{id}/b/{idB}    creeaza legatura: 201 daca e noua, 204 daca exista deja (idempotent)
//   DELETE /a/{id}/b/{idB}    desface legatura: 204; resursele raman (doar legatura dispare)
//
// PUT nu are corp: adresa spune tot ce trebuie legat de ce.

// ---------- scripturile unui calculator: /computers/:id/scripts ----------

func computerScriptsPath(c *gin.Context) string {
	return computersPath + "/" + c.Param("id") + "/scripts"
}

// listScripts: GET /computers/:id/scripts[?isActive=true|false][&verbose] (L3 2.1).
// Fara verbose, fiecare script apare doar cu id-ul si link-urile; cu verbose apar si numele
// si caile de stocare.
func (h *computerHandler) listScripts(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	isActive, ok := httpx.QueryBool(c, "isActive")
	if !ok {
		return
	}
	verbose, ok := httpx.QueryFlag(c, "verbose")
	if !ok {
		return
	}
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	scripts, total, err := h.svc.ComputerScripts(c.Request.Context(), identity, id, isActive, page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	subPath := computerScriptsPath(c)
	items := make([]any, 0, len(scripts))
	for _, script := range scripts {
		links := relationLinks(subPath, script.ID, "script", scriptsPath, identity)
		if verbose {
			items = append(items, dto.NewScript(script, links))
		} else {
			items = append(items, dto.Ref{ID: script.ID.Hex(), Links: links})
		}
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addSubCollectionLinks(body.Links, computersPath+"/"+id.Hex(), subPath, "scriptId", identity)
	c.JSON(http.StatusOK, body)
}

func (h *computerHandler) getScript(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	scriptID, ok := httpx.ParseID(c, "scriptId")
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	script, err := h.svc.ComputerScript(c.Request.Context(), identity, id, scriptID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewScript(script, relationLinks(computerScriptsPath(c), scriptID, "script", scriptsPath, identity)))
}

// installScript: PUT /computers/:id/scripts/:scriptId (SP 1.3: "instaleaza un script").
func (h *computerHandler) installScript(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	scriptID, ok := httpx.ParseID(c, "scriptId")
	if !ok {
		return
	}
	created, err := h.svc.InstallScript(c.Request.Context(), id, scriptID)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	// 201: intoarcem scriptul, vazut de pe calculator.
	identity := auth.FromContext(c)
	script, err := h.svc.ComputerScript(c.Request.Context(), identity, id, scriptID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewScript(script, relationLinks(computerScriptsPath(c), scriptID, "script", scriptsPath, identity)))
}

// uninstallScript: DELETE /computers/:id/scripts/:scriptId (SP 1.3: "revoca un script").
func (h *computerHandler) uninstallScript(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	scriptID, ok := httpx.ParseID(c, "scriptId")
	if !ok {
		return
	}
	if err := h.svc.UninstallScript(c.Request.Context(), id, scriptID); err != nil {
		abortError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- calculatoarele unui cluster: /clusters/:id/computers ----------

func clusterComputersPath(c *gin.Context) string {
	return clustersPath + "/" + c.Param("id") + "/computers"
}

func (h *clusterHandler) listComputers(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	computers, total, err := h.svc.ClusterComputers(c.Request.Context(), identity, id, page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	subPath := clusterComputersPath(c)
	items := make([]dto.Computer, 0, len(computers))
	for _, computer := range computers {
		items = append(items, dto.NewComputer(computer, relationLinks(subPath, computer.ID, "computer", computersPath, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addSubCollectionLinks(body.Links, clustersPath+"/"+id.Hex(), subPath, "computerId", identity)
	c.JSON(http.StatusOK, body)
}

func (h *clusterHandler) getComputer(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	computerID, ok := httpx.ParseID(c, "computerId")
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	computer, err := h.svc.ClusterComputer(c.Request.Context(), identity, id, computerID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewComputer(computer, relationLinks(clusterComputersPath(c), computerID, "computer", computersPath, identity)))
}

// addComputer: PUT /clusters/:id/computers/:computerId (SP 1.3: "adauga un calculator in cluster").
// 409 daca acel calculator e deja in alt cluster.
func (h *clusterHandler) addComputer(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	computerID, ok := httpx.ParseID(c, "computerId")
	if !ok {
		return
	}
	created, err := h.svc.AddComputerToCluster(c.Request.Context(), id, computerID)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	identity := auth.FromContext(c)
	computer, err := h.svc.ClusterComputer(c.Request.Context(), identity, id, computerID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewComputer(computer, relationLinks(clusterComputersPath(c), computerID, "computer", computersPath, identity)))
}

// removeComputer: DELETE /clusters/:id/computers/:computerId (SP 1.3: "elimina un calculator din cluster").
func (h *clusterHandler) removeComputer(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	computerID, ok := httpx.ParseID(c, "computerId")
	if !ok {
		return
	}
	if err := h.svc.RemoveComputerFromCluster(c.Request.Context(), id, computerID); err != nil {
		abortError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
