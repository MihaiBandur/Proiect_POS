package memory

import "slices"

// addTo adauga v in lista daca nu exista deja ($addToSet in MongoDB) si spune daca a adaugat.
// slices.Clip obliga append sa aloce o lista noua: documentele intoarse mai devreme de Get
// impart memoria cu cele salvate, iar o scriere pe loc ar modifica si copiile lor.
func addTo[E comparable](list *[]E, v E) bool {
	if slices.Contains(*list, v) {
		return false
	}
	*list = append(slices.Clip(*list), v)
	return true
}

// removeFrom scoate v din lista ($pull in MongoDB) si spune daca l-a gasit.
// Lucreaza pe o copie, din acelasi motiv ca addTo.
func removeFrom[E comparable](list *[]E, v E) bool {
	i := slices.Index(*list, v)
	if i < 0 {
		return false
	}
	*list = slices.Delete(slices.Clone(*list), i, i+1)
	return true
}
