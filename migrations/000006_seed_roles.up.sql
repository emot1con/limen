INSERT INTO roles (code, name) VALUES
    ('siswa', 'Siswa'),
    ('guru_mapel', 'Guru Mapel'),
    ('guru_wali', 'Guru Wali'),
    ('guru_bk', 'Guru BK'),
    ('admin', 'Admin')
ON CONFLICT (code) DO NOTHING;
