package handler

import (
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/httpx"
)

// itemLinks: self si parent pentru oricine (L3, pag. 3); actiunile doar pentru admin (L3 1.4).
func itemLinks(collectionPath string, id bson.ObjectID, identity auth.Identity) httpx.Links {
	self := collectionPath + "/" + id.Hex()
	links := httpx.Links{
		"self":   {Href: self},
		"parent": {Href: collectionPath},
	}
	if identity.IsAdmin() {
		links["edit"] = httpx.Link{Href: self, Type: http.MethodPut}
		links["delete"] = httpx.Link{Href: self, Type: http.MethodDelete}
	}
	return links
}

// addCollectionLinks completeaza link-urile unei colectii: parent e punctul de intrare,
// iar create (POST) apare doar pentru admin.
func addCollectionLinks(links httpx.Links, collectionPath string, identity auth.Identity) {
	links["parent"] = httpx.Link{Href: basePath}
	if identity.IsAdmin() {
		links["create"] = httpx.Link{Href: collectionPath, Type: http.MethodPost}
	}
}

// deletedLinks: dupa DELETE resursa nu mai exista, deci ramane doar drumul inapoi la colectie.
func deletedLinks(collectionPath string) httpx.Links {
	return httpx.Links{"parent": {Href: collectionPath}}
}

// Link-uri pentru sub-colectii (rutele de legatura din L3 2.1, ex. /computers/{id}/scripts).

// relationLinks sunt link-urile unui element dintr-o sub-colectie, de exemplu un script
// instalat pe un calculator:
//   - self e legatura insasi (/computers/{id}/scripts/{scriptId});
//   - parent e sub-colectia (/computers/{id}/scripts);
//   - rel duce la resursa principala (ex. "script": /scripts/{scriptId});
//   - adminul primeste delete, care desface legatura, dar nu sterge resursa.
func relationLinks(subPath string, targetID bson.ObjectID, rel, targetCollection string, identity auth.Identity) httpx.Links {
	self := subPath + "/" + targetID.Hex()
	links := httpx.Links{
		"self":   {Href: self},
		"parent": {Href: subPath},
		rel:      {Href: targetCollection + "/" + targetID.Hex()},
	}
	if identity.IsAdmin() {
		links["delete"] = httpx.Link{Href: self, Type: http.MethodDelete}
	}
	return links
}

// addSubCollectionLinks: parent e resursa care contine lista; adminul primeste si un sablon
// (RFC 6570) pentru adaugarea unei legaturi, ex. PUT /computers/{id}/scripts/{scriptId}.
func addSubCollectionLinks(links httpx.Links, ownerPath, subPath, param string, identity auth.Identity) {
	links["parent"] = httpx.Link{Href: ownerPath}
	if identity.IsAdmin() {
		links["add"] = httpx.Link{Href: subPath + "/{" + param + "}", Type: http.MethodPut, Templated: true}
	}
}

// computerLinks: link-urile unui calculator, plus navigarea catre sub-colectiile lui.
func computerLinks(id bson.ObjectID, identity auth.Identity) httpx.Links {
	links := itemLinks(computersPath, id, identity)
	self := links["self"].Href
	links["scripts"] = httpx.Link{Href: self + "/scripts"}
	if identity.IsAdmin() {
		links["users"] = httpx.Link{Href: self + "/users"}
	}
	return links
}

// clusterLinks: link-urile unui cluster, plus navigarea catre sub-colectiile lui.
func clusterLinks(id bson.ObjectID, identity auth.Identity) httpx.Links {
	links := itemLinks(clustersPath, id, identity)
	self := links["self"].Href
	links["computers"] = httpx.Link{Href: self + "/computers"}
	if identity.IsAdmin() {
		links["users"] = httpx.Link{Href: self + "/users"}
	}
	if identity.IsMaintainer() {
		// Sablon (RFC 6570): maintainerul ruleaza un script pe tot clusterul (SP 1.3).
		links["run"] = httpx.Link{Href: self + "/scripts/{scriptId}/tasks", Type: http.MethodPost, Templated: true}
	}
	return links
}
