"use client";

import { usePipelineStore } from "../../model/pipeline.store";
import JudgmentEditor from "../JudgmentEditor";

export default function JudgmentSection() {
  const successScript = usePipelineStore((s) => s.successScript);
  const failureScript = usePipelineStore((s) => s.failureScript);
  const setSuccessScript = usePipelineStore((s) => s.setSuccessScript);
  const setFailureScript = usePipelineStore((s) => s.setFailureScript);

  return (
    <div className="border border-nexus-border rounded-lg p-4">
      <h3 className="text-sm font-semibold text-nexus-text-primary uppercase tracking-wider mb-3">
        Judgment
      </h3>

      <JudgmentEditor
        successScript={successScript}
        failureScript={failureScript}
        onSuccessChange={setSuccessScript}
        onFailureChange={setFailureScript}
      />
    </div>
  );
}
