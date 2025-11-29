import { Show, JSX } from "solid-js";

export interface PaneProps {
  id: string;
  title: string;
  collapsed: boolean;
  collapsedStatus?: () => string | JSX.Element | null;
  onToggle: () => void;
  // Drag-and-drop
  onDragStart: () => void;
  onDragOver: () => void;
  onDrop: () => void;
  isDragTarget: boolean;
  children: JSX.Element;
}

export function Pane(props: PaneProps) {
  return (
    <div
      class="tygor-pane"
      classList={{
        "tygor-pane--collapsed": props.collapsed,
        "tygor-pane--drag-target": props.isDragTarget,
      }}
      draggable={true}
      ondragstart={(e) => {
        e.dataTransfer!.effectAllowed = "move";
        props.onDragStart();
      }}
      ondragover={(e) => {
        e.preventDefault();
        props.onDragOver();
      }}
      ondrop={(e) => {
        e.preventDefault();
        props.onDrop();
      }}
    >
      <div class="tygor-pane-header" onClick={props.onToggle}>
        <span class="tygor-pane-chevron">{props.collapsed ? "▸" : "▾"}</span>
        <span class="tygor-pane-title">{props.title}</span>
        <Show when={props.collapsed && props.collapsedStatus}>
          <span class="tygor-pane-status">{props.collapsedStatus!()}</span>
        </Show>
      </div>
      <Show when={!props.collapsed}>
        <div class="tygor-pane-content">
          {props.children}
        </div>
      </Show>
    </div>
  );
}
