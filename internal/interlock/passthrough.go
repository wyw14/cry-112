package interlock

import (
	"fmt"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type PassThroughDecision struct {
	RequestedDoor string    `json:"requested_door"`
	PeerDoor      string    `json:"peer_door"`
	PeerProof     DoorProof `json:"peer_proof"`
	Permit        bool      `json:"permit"`
	Reason        string    `json:"reason"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
}

type PassThrough struct {
	evaluator DoorEvaluator
}

func NewPassThrough() PassThrough {
	return PassThrough{evaluator: NewDoorEvaluator()}
}

func (p PassThrough) EvaluatePeer(requested model.DoorState, peer model.DoorState, now time.Time) (PassThroughDecision, error) {
	if requested.ID == "" || peer.ID == "" || requested.ID == peer.ID {
		return PassThroughDecision{}, fmt.Errorf("distinct requested and peer doors are required")
	}
	proof := p.evaluator.Evaluate(peer, now)
	decision := PassThroughDecision{RequestedDoor: requested.ID, PeerDoor: peer.ID, PeerProof: proof, EvaluatedAt: now.UTC()}
	if !proof.Valid() {
		decision.Reason = "opposite door physical proof is incomplete"
		return decision, nil
	}
	decision.Permit = true
	decision.Reason = "opposite door is physically closed, locked and sealed"
	return decision, nil
}
