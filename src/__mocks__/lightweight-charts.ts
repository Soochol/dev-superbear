const createMockSeries = () => ({
  setData: jest.fn(),
  update: jest.fn(),
  applyOptions: jest.fn(),
});

export const createChart = jest.fn(() => ({
  addSeries: jest.fn(() => createMockSeries()),
  removeSeries: jest.fn(),
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
