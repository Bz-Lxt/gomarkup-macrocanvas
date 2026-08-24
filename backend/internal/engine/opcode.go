package engine

type OpKind uint8

const (
	OpNop OpKind = iota
	OpKeyDown
	OpKeyUp
	OpMouseRel
	OpMouseAbs
	OpMouseBtn
	OpMouseWheel
	OpWait
	OpWaitRand
	OpJump
	OpJumpIf
	OpVarSet
	OpVarInc
	OpMarker
	OpBreak
	OpHalt
)

type CondKind uint8

const (
	CondAlways CondKind = iota
	CondLoopGE
	CondLoopLT
	CondVarGE
	CondVarLT
	CondElapsedGE
	CondKeyDown
)

type Op struct {
	Kind    OpKind
	Page    uint16
	Usage   uint16
	Value   int32
	DelayNs int64
	Jump    int32
	Cond    CondKind
	VarID   int32
	Imm     int64
	NodeID  string
	Label   string
}

type Program struct {
	Ops       []Op
	Precision string
	MaxIters  int
	MaxWallMs int
	SourceIDs []string
}
