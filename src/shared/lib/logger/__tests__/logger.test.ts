import { logger } from "../index";

describe("Logger", () => {
  beforeEach(() => {
    jest.spyOn(console, "debug").mockImplementation(() => {});
    jest.spyOn(console, "info").mockImplementation(() => {});
    jest.spyOn(console, "warn").mockImplementation(() => {});
    jest.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("logs debug messages in development", () => {
    logger.debug("test message");
    expect(console.debug).toHaveBeenCalled();
  });

  it("logs info messages", () => {
    logger.info("test info");
    expect(console.info).toHaveBeenCalled();
  });

  it("logs warn messages", () => {
    logger.warn("test warn");
    expect(console.warn).toHaveBeenCalled();
  });

  it("logs error messages", () => {
    logger.error("test error");
    expect(console.error).toHaveBeenCalled();
  });

  it("includes timestamp in output", () => {
    logger.info("timestamped");
    const call = (console.info as jest.Mock).mock.calls[0][0];
    expect(call).toMatch(/\[\d{4}-\d{2}-\d{2}T/);
  });

  it("includes level in output", () => {
    logger.error("level check");
    const call = (console.error as jest.Mock).mock.calls[0][0];
    expect(call).toContain("[ERROR]");
  });

  it("includes context when provided", () => {
    logger.info("with context", { userId: "123" });
    const call = (console.info as jest.Mock).mock.calls[0][0];
    expect(call).toContain('"userId":"123"');
  });
});
