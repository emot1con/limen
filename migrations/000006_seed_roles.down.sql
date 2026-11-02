DELETE FROM roles
WHERE code IN ('siswa', 'guru_mapel', 'guru_wali', 'guru_bk', 'admin')
  AND NOT EXISTS (SELECT 1 FROM user_roles WHERE user_roles.role_code = roles.code);
