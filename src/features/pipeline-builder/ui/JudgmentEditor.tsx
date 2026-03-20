"use client";

import DSLConditionEditor from "./DSLConditionEditor";
import PriceAlertEditor from "./PriceAlertEditor";

interface JudgmentEditorProps {
  successScript: string;
  failureScript: string;
  onSuccessChange: (value: string) => void;
  onFailureChange: (value: string) => void;
}

export default function JudgmentEditor({
  successScript,
  failureScript,
  onSuccessChange,
  onFailureChange,
}: JudgmentEditorProps) {
  return (
    <div className="space-y-4">
      <div className="flex gap-4">
        <DSLConditionEditor
          label="Success Condition"
          value={successScript}
          onChange={onSuccessChange}
          accent="success"
          placeholder="e.g. confidence > 0.7 AND risk < 0.3"
        />
        <DSLConditionEditor
          label="Failure Condition"
          value={failureScript}
          onChange={onFailureChange}
          accent="failure"
          placeholder="e.g. confidence < 0.3 OR risk > 0.8"
        />
      </div>

      <PriceAlertEditor />
    </div>
  );
}
