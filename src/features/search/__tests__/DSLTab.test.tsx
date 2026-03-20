import { render, screen } from "@testing-library/react";
import { DSLTab } from "../ui/DSLTab";
import { useSearchStore } from "../model/search.store";

// Mock DSLEditor since CodeMirror requires real browser DOM APIs
jest.mock("../ui/DSLEditor", () => ({
  DSLEditor: ({ placeholder }: { placeholder?: string }) => (
    <div data-testid="dsl-editor-container">{placeholder}</div>
  ),
}));

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("DSLTab", () => {
  it("renders the DSL editor area", () => {
    render(<DSLTab />);
    expect(screen.getByTestId("dsl-editor-container")).toBeInTheDocument();
  });

  it("renders Validate button", () => {
    render(<DSLTab />);
    expect(
      screen.getByRole("button", { name: /validate/i }),
    ).toBeInTheDocument();
  });

  it("renders Explain in NL button", () => {
    render(<DSLTab />);
    expect(
      screen.getByRole("button", { name: /explain/i }),
    ).toBeInTheDocument();
  });

  it("renders Run Search button", () => {
    render(<DSLTab />);
    expect(
      screen.getByRole("button", { name: /run|실행/i }),
    ).toBeInTheDocument();
  });
});
