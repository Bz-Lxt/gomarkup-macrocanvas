export type Precision = "realtime" | "balanced" | "efficient";

export interface NodeModel {
  id: string;
  type: string;
  x: number;
  y: number;
  params: Record<string, unknown>;
}

export interface EdgeModel {
  id: string;
  from: string;
  to: string;
  port: string;
}

export interface MacroModel {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  deployed: boolean;
  precision: Precision;
  trigger: { kind: string; hotkey: string };
  nodes: NodeModel[];
  edges: EdgeModel[];
  budget: { max_iters: number; max_wall_ms: number; watchdog_ms: number };
  created_at: string;
  updated_at: string;
  version: number;
}

export const PALETTE = [
  { type: "key.tap", label: "点按按键", group: "键" },
  { type: "key.down", label: "按下", group: "键" },
  { type: "key.up", label: "松开", group: "键" },
  { type: "key.combo", label: "组合键", group: "键" },
  { type: "text.type", label: "输入文本", group: "键" },
  { type: "mouse.move.rel", label: "相对平移", group: "鼠" },
  { type: "mouse.click", label: "点击", group: "鼠" },
  { type: "mouse.scroll", label: "滚轮", group: "鼠" },
  { type: "mouse.drag", label: "拖拽", group: "鼠" },
  { type: "wait.fixed", label: "固定等待", group: "时" },
  { type: "wait.random", label: "随机等待", group: "时" },
  { type: "flow.loop", label: "循环", group: "流" },
  { type: "flow.if", label: "条件分支", group: "流" },
  { type: "flow.break", label: "跳出", group: "流" },
  { type: "debug.marker", label: "标记", group: "流" },
];
