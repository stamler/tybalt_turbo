export type NoteAuthor = {
  id: string;
  email: string;
  given_name: string;
  surname: string;
};

export type NoteJob = {
  id: string;
  number?: string | null;
  description?: string | null;
};

export type ClientNote = {
  id: string;
  created: string;
  note: string;
  job: NoteJob | null;
  author: NoteAuthor;
};

export type NoteJobOption = {
  id: string;
  number?: string | null;
  description?: string | null;
};
