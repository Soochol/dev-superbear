export const createChart = jest.fn(() => ({
  addSeries: jest.fn(),
  applyOptions: jest.fn(),
  remove: jest.fn(),
  timeScale: jest.fn(() => ({
    fitContent: jest.fn(),
    subscribeVisibleLogicalRangeChange: jest.fn(),
  })),
  subscribeCrosshairMove: jest.fn(),
}));

export const CandlestickSeries = {};
export const LineSeries = {};
export const HistogramSeries = {};
