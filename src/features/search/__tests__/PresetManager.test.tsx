/** @jest-environment jsdom */
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PresetManager } from "../ui/PresetManager";
import { usePresetStore } from "../model/preset.store";
import { useSearchStore } from "../model/search.store";
import { presetApi } from "../api/preset-api";

jest.mock("../api/preset-api", () => ({
  presetApi: {
    list: jest.fn(),
    create: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockedApi = presetApi as jest.Mocked<typeof presetApi>;

beforeEach(() => {
  usePresetStore.setState(usePresetStore.getInitialState());
  useSearchStore.setState(useSearchStore.getInitialState());
  jest.clearAllMocks();
  mockedApi.list.mockResolvedValue({ data: [], pagination: { total: 0, page: 1, pageSize: 20, totalPages: 0 } });
});

describe("PresetManager", () => {
  it("renders Save Preset button", () => {
    render(<PresetManager />);
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("shows saved presets from store", () => {
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "My Preset", dsl: "scan where volume > 100", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    expect(screen.getByText("My Preset")).toBeInTheDocument();
  });

  it("clicking a preset loads its DSL into the editor", () => {
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "Volume Filter", dsl: "scan where volume > 5000000", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    fireEvent.click(screen.getByText("Volume Filter"));
    expect(useSearchStore.getState().dslCode).toBe("scan where volume > 5000000");
  });

  it("delete button removes preset", async () => {
    mockedApi.delete.mockResolvedValue(undefined);
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "To Delete", dsl: "scan", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    fireEvent.click(screen.getByLabelText("Delete preset To Delete"));
    await waitFor(() => {
      expect(mockedApi.delete).toHaveBeenCalledWith("1");
    });
  });
});
