import { render, screen, fireEvent } from "@testing-library/react";
import { NLTab } from "../ui/NLTab";
import { useSearchStore } from "../model/search.store";

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("NLTab", () => {
  it("renders textarea for NL query input", () => {
    render(<NLTab />);
    expect(screen.getByPlaceholderText(/자연어로 검색 조건/i)).toBeInTheDocument();
  });

  it("renders preset chips", () => {
    render(<NLTab />);
    expect(screen.getByText("2yr Max Volume")).toBeInTheDocument();
    expect(screen.getByText("Golden Cross")).toBeInTheDocument();
    expect(screen.getByText("RSI Oversold")).toBeInTheDocument();
  });

  it("clicking a preset chip fills the NL query", () => {
    render(<NLTab />);
    fireEvent.click(screen.getByText("2yr Max Volume"));
    const textarea = screen.getByPlaceholderText(/자연어로 검색 조건/i) as HTMLTextAreaElement;
    expect(textarea.value).toContain("2년 최대거래량");
  });

  it("shows agent status when searching", () => {
    useSearchStore.setState({ agentStatus: "interpreting" });
    render(<NLTab />);
    expect(screen.getByText(/Interpreting/i)).toBeInTheDocument();
  });

  it("has a search button", () => {
    render(<NLTab />);
    expect(screen.getByRole("button", { name: /검색|search/i })).toBeInTheDocument();
  });
});
