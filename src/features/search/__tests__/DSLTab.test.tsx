/** @jest-environment jsdom */
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DSLTab } from "../ui/DSLTab";
import { useSearchStore } from "../model/search.store";

import { searchApi } from "../api/search-api";

jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;

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

  it("calls validate API when Validate button is clicked", async () => {
    mockedApi.validate.mockResolvedValue({ valid: true, error: null });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);
    fireEvent.click(screen.getByRole("button", { name: /validate/i }));
    await waitFor(() => {
      expect(mockedApi.validate).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });

  it("calls explain API when Explain button is clicked", async () => {
    mockedApi.explain.mockResolvedValue({ explanation: "test explanation" });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);
    fireEvent.click(screen.getByRole("button", { name: /explain/i }));
    await waitFor(() => {
      expect(mockedApi.explain).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });

  it("displays explanation text after Explain button is clicked", async () => {
    mockedApi.explain.mockResolvedValue({ explanation: "거래량 100만 이상 종목 검색" });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);
    fireEvent.click(screen.getByRole("button", { name: /explain/i }));
    await waitFor(() => {
      expect(screen.getByText("거래량 100만 이상 종목 검색")).toBeInTheDocument();
    });
  });

  it("calls dslSearch API when Run Search button is clicked", async () => {
    mockedApi.dslSearch.mockResolvedValue({ results: [] });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);
    fireEvent.click(screen.getByRole("button", { name: /run/i }));
    await waitFor(() => {
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
});
