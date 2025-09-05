package storage

type BadgerStorage struct{}

// OpenDatabase will handle logic for opening and setting up databse
func (s *BadgerStorage) Open(path string) error {
	return nil
}

// Close will handle closing and cleaning up database instance
func (s *BadgerStorage) Close() error {
	return nil
}

// Put will set a key to a value that is passed in.
func (s *BadgerStorage) Put(key string, value map[string]any) error {
	return nil
}

// Get will retrieve the value of the supplied key
func (s *BadgerStorage) Get(key string) ([]byte, error) {
	return nil, nil
}

// Delete will delete a key, value pair from the database.
func (s *BadgerStorage) Delete(key string) error {
	return nil
}
