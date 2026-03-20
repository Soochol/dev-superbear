export interface SearchPreset {
  id: string;
  userId: string;
  name: string;
  dsl: string;
  nlQuery: string | null;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateSearchPresetInput {
  name: string;
  dsl: string;
  nlQuery?: string;
  isPublic?: boolean;
}
