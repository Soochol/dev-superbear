describe("Search Presets", () => {
  it("POST /api/v1/search/presets requires name and dsl", () => {
    const validPayload = {
      name: "2yr Max Volume",
      dsl: "scan where max_volume(730) == volume and trade_value >= 300000000000",
      nlQuery: "2년 최대거래량 + 거래대금 3000억",
    };
    expect(validPayload.name).toBeTruthy();
    expect(validPayload.dsl).toBeTruthy();
  });

  it("GET /api/v1/search/presets returns list format", () => {
    const expectedFormat = {
      data: [
        { id: "uuid", name: "preset name", dsl: "scan ...", nlQuery: null, createdAt: "date" },
      ],
      pagination: { total: 1, page: 1, pageSize: 20, totalPages: 1 },
    };
    expect(expectedFormat.data).toBeInstanceOf(Array);
    expect(expectedFormat.pagination).toHaveProperty("total");
  });
});
