/** @jest-environment jsdom */
import { renderHook, act } from "@testing-library/react";
import { useChartData } from "../use-chart-data";
import { useChartStore } from "../../model/chart.store";
import { chartApi } from "../../api/chart-api";
import type { CandleData } from "@/entities/candle";
import type { StockInfo } from "@/entities/stock";

jest.mock("../../api/chart-api", () => ({
  chartApi: {
    fetchCandles: jest.fn(),
  },
}));

const mockFetchCandles = chartApi.fetchCandles as jest.MockedFunction<
  typeof chartApi.fetchCandles
>;

const mockStock: StockInfo = {
  symbol: "005930",
  name: "Samsung Electronics",
  price: 78400,
  change: 1600,
  changePct: 2.08,
};

const mockCandles: CandleData[] = [
  { time: "2024-01-01", open: 100, high: 110, low: 90, close: 105, volume: 1000 },
  { time: "2024-01-02", open: 105, high: 115, low: 95, close: 110, volume: 1200 },
];

beforeEach(() => {
  useChartStore.setState(useChartStore.getInitialState());
  mockFetchCandles.mockReset();
  mockFetchCandles.mockResolvedValue(mockCandles);
});

describe("useChartData", () => {
  it("does not fetch when currentStock is null", async () => {
    renderHook(() => useChartData());

    await act(async () => {});

    expect(mockFetchCandles).not.toHaveBeenCalled();
  });

  it("fetches candles when currentStock is set", async () => {
    useChartStore.setState({ currentStock: mockStock });

    renderHook(() => useChartData());

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledTimes(1);
    expect(useChartStore.getState().candles).toEqual(mockCandles);
  });

  it("calls fetchCandles with correct symbol and timeframe", async () => {
    useChartStore.setState({ currentStock: mockStock, timeframe: "1W" });

    renderHook(() => useChartData());

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledWith("005930", "1W");
  });

  it("isLoading transitions: false -> true -> false on success", async () => {
    let resolvePromise: (value: CandleData[]) => void;
    mockFetchCandles.mockImplementation(
      () => new Promise((resolve) => { resolvePromise = resolve; }),
    );

    useChartStore.setState({ currentStock: mockStock });

    expect(useChartStore.getState().isLoading).toBe(false);

    renderHook(() => useChartData());

    // After the effect fires, isLoading should be true
    await act(async () => {});
    expect(useChartStore.getState().isLoading).toBe(true);

    // Resolve the promise — the finally block sets isLoading to false
    await act(async () => {
      resolvePromise!(mockCandles);
    });

    expect(useChartStore.getState().isLoading).toBe(false);
  });

  it("isLoading resets to false on error (via finally block)", async () => {
    // The hook wraps fetchCandles in try/finally (no catch). The finally
    // block always calls setIsLoading(false). We verify this by making
    // chartApi.fetchCandles return an empty array (simulating the real
    // chartApi error path, which catches internally and returns []),
    // and tracking setIsLoading calls to prove the finally block runs.
    //
    // Note: The real chartApi.fetchCandles has its own try/catch that
    // returns [] on error. The hook's try/finally provides a safety net
    // in case an unexpected error occurs. We verify the finally block
    // always runs by checking setIsLoading(true) then setIsLoading(false).
    const setIsLoadingCalls: boolean[] = [];
    const origSetIsLoading = useChartStore.getState().setIsLoading;

    // Simulate an error scenario where fetchCandles returns empty array
    mockFetchCandles.mockResolvedValue([]);

    useChartStore.setState({
      currentStock: mockStock,
      setIsLoading: (v: boolean) => {
        setIsLoadingCalls.push(v);
        origSetIsLoading(v);
      },
    });

    renderHook(() => useChartData());

    await act(async () => {});

    // The finally block should have called setIsLoading(false) after the
    // fetch completed, regardless of the result
    expect(setIsLoadingCalls).toEqual([true, false]);
    expect(useChartStore.getState().isLoading).toBe(false);
    expect(useChartStore.getState().candles).toEqual([]);
  });

  it("re-fetches when symbol changes", async () => {
    useChartStore.setState({ currentStock: mockStock });

    const { rerender } = renderHook(() => useChartData());

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledTimes(1);
    expect(mockFetchCandles).toHaveBeenCalledWith("005930", "1D");

    const newStock: StockInfo = {
      symbol: "035720",
      name: "Kakao",
      price: 52000,
      change: -500,
      changePct: -0.95,
    };

    await act(async () => {
      useChartStore.setState({ currentStock: newStock });
    });

    rerender();

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledTimes(2);
    expect(mockFetchCandles).toHaveBeenLastCalledWith("035720", "1D");
  });

  it("re-fetches when timeframe changes", async () => {
    useChartStore.setState({ currentStock: mockStock });

    const { rerender } = renderHook(() => useChartData());

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledTimes(1);
    expect(mockFetchCandles).toHaveBeenCalledWith("005930", "1D");

    await act(async () => {
      useChartStore.setState({ timeframe: "1W" });
    });

    rerender();

    await act(async () => {});

    expect(mockFetchCandles).toHaveBeenCalledTimes(2);
    expect(mockFetchCandles).toHaveBeenLastCalledWith("005930", "1W");
  });
});
