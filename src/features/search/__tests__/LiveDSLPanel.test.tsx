/** @jest-environment jsdom */
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { LiveDSLPanel } from "../ui/LiveDSLPanel";
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

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("LiveDSLPanel", () => {
  it("renders the LIVE DSL label", () => {
    render(<LiveDSLPanel />);
    expect(screen.getByText(/LIVE DSL/i)).toBeInTheDocument();
  });

  it("shows DSL code from store", () => {
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/scan/)).toBeInTheDocument();
    expect(screen.getByText(/volume/)).toBeInTheDocument();
  });

  it("shows empty state when no DSL", () => {
    render(<LiveDSLPanel />);
    expect(screen.getByText(/DSL이 없습니다/i)).toBeInTheDocument();
  });

  it("shows validation badge when validated", () => {
    useSearchStore.setState({
      dslCode: "scan where volume > 1000000",
      validationState: "valid",
    });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/validated/i)).toBeInTheDocument();
  });

  it("shows warning badge when not validated", () => {
    useSearchStore.setState({
      dslCode: "scan where volume > 1000000",
      validationState: "none",
    });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/not validated/i)).toBeInTheDocument();
  });

  it("renders Copy and Run Search buttons", () => {
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run/i })).toBeInTheDocument();
  });

  it("calls dslSearch when Run button is clicked", async () => {
    mockedApi.dslSearch.mockResolvedValue({ results: [] });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);
    fireEvent.click(screen.getByRole("button", { name: /run/i }));
    await waitFor(() => {
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
});
