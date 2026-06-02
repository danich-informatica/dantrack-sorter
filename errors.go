package sorter

import "errors"

// Errores sentinel de la librería dantrack-sorter.
//
// Regla: los errores Go se usan SOLO para problemas que impiden producir una
// decisión (config inválida, request inválido, ambigüedad no resoluble).
//
// ActionReject es una decisión válida y NO devuelve error.
// La ausencia de destino válido produce ActionReject con trazabilidad, no un error.

var (
	// ErrInvalidConfig se devuelve cuando la configuración del engine es inválida
	// y no permite operar. Verificar con errors.Is(err, ErrInvalidConfig).
	ErrInvalidConfig = errors.New("invalid engine configuration")

	// ErrInvalidRequest se devuelve cuando el request no cumple los requisitos mínimos.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMissingRequiredField se devuelve cuando falta un campo obligatorio
	// en la config o en el request.
	ErrMissingRequiredField = errors.New("missing required field")

	// ErrNoAssignmentMatch se devuelve si no existe ninguna asignación que aplique
	// y no se puede producir una decisión válida (ni siquiera ActionReject).
	// En la mayoría de los casos, la ausencia de match produce ActionReject, no este error.
	ErrNoAssignmentMatch = errors.New("no assignment matches box context")

	// ErrNoAvailableExit se devuelve si la configuración no define ninguna salida
	// (ni siquiera default/reject) y la librería no puede producir decisión alguna.
	ErrNoAvailableExit = errors.New("no available exit")

	// ErrNoAvailablePark se devuelve si la configuración no define ningún parque
	// (ni siquiera default) y la librería no puede producir decisión alguna.
	ErrNoAvailablePark = errors.New("no available park")

	// ErrAmbiguousAssignment se devuelve cuando múltiples asignaciones con la misma
	// prioridad apuntan a destinos diferentes y la política es error (no first-wins).
	ErrAmbiguousAssignment = errors.New("ambiguous assignment: multiple matches with equal priority pointing to different targets")

	// ErrContextCanceled se devuelve cuando el context.Context fue cancelado antes
	// de completar la decisión.
	ErrContextCanceled = errors.New("context canceled")
)
