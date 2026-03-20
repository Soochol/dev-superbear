"use client";

import { useEffect, useRef } from "react";
import {
  EditorView,
  keymap,
  placeholder as phExtension,
} from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap } from "@codemirror/commands";
import {
  autocompletion,
  type CompletionContext,
} from "@codemirror/autocomplete";
import { DSL_COMPLETIONS } from "../lib/dsl-completions";
import { useSearchStore } from "../model/search.store";

const nexusDarkTheme = EditorView.theme({
  "&": {
    backgroundColor: "#0a0a0f",
    color: "#e2e8f0",
    fontSize: "14px",
  },
  ".cm-content": {
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    padding: "12px",
  },
  ".cm-gutters": {
    backgroundColor: "#12121a",
    borderRight: "1px solid #1e1e2e",
  },
  ".cm-activeLine": { backgroundColor: "rgba(99, 102, 241, 0.08)" },
  ".cm-cursor": { borderLeftColor: "#6366f1" },
  "&.cm-focused .cm-selectionBackground": {
    backgroundColor: "rgba(99, 102, 241, 0.2)",
  },
});

function dslAutoComplete(context: CompletionContext) {
  const word = context.matchBefore(/\w*/);
  if (!word || (word.from === word.to && !context.explicit)) return null;
  return {
    from: word.from,
    options: DSL_COMPLETIONS.map((c) => ({
      label: c.label,
      type: c.type,
      detail: c.detail,
    })),
  };
}

interface DSLEditorProps {
  readOnly?: boolean;
  placeholder?: string;
  height?: string;
}

export function DSLEditor({
  readOnly = false,
  placeholder = "",
  height = "200px",
}: DSLEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const { dslCode, setDslCode } = useSearchStore();

  useEffect(() => {
    if (!containerRef.current) return;

    const state = EditorState.create({
      doc: dslCode,
      extensions: [
        keymap.of(defaultKeymap),
        nexusDarkTheme,
        EditorView.lineWrapping,
        phExtension(placeholder),
        autocompletion({ override: [dslAutoComplete] }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !readOnly) {
            setDslCode(update.state.doc.toString());
          }
        }),
        ...(readOnly ? [EditorState.readOnly.of(true)] : []),
      ],
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => {
      view.destroy();
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync when dslCode changes externally (e.g., NL agent generates DSL)
  useEffect(() => {
    const view = viewRef.current;
    if (view && view.state.doc.toString() !== dslCode) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: dslCode },
      });
    }
  }, [dslCode]);

  return (
    <div
      ref={containerRef}
      data-testid="dsl-editor-container"
      className="border border-nexus-border rounded-lg overflow-hidden"
      style={{ minHeight: height }}
    />
  );
}
