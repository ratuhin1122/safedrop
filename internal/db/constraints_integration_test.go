package db

import "testing"

func TestIncomingAndFinalConstraints(t *testing.T) {
	for _, key := range []string{"final/unscanned", "public/file"} {
		t.Run(key, func(t *testing.T) {
			ctx, q := testQueries(t)
			u, err := q.CreateUser(ctx, CreateUserParams{Email: "constraint@example.test", PasswordHash: "hash"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = q.CreateFile(ctx, CreateFileParams{OwnerID: u.ID, OriginalName: "file",
				Mime: "application/octet-stream", Size: 1, StorageKey: key})
			if err == nil {
				t.Fatal("accepted uploaded file outside incoming/")
			}
		})
	}
}
