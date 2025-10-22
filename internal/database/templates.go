package database

import (
	"database/sql"
	"fmt"

	"wepoc/internal/models"
)

// InsertTemplate inserts a new template into the database
func (d *Database) InsertTemplate(template *models.Template) error {
	query := `
		INSERT INTO templates (template_id, name, severity, tags, author, file_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query,
		template.TemplateID,
		template.Name,
		template.Severity,
		template.Tags,
		template.Author,
		template.FilePath,
	)
	if err != nil {
		return fmt.Errorf("failed to insert template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	template.ID = id
	return nil
}

// GetTemplateByID retrieves a template by its database ID
func (d *Database) GetTemplateByID(id int64) (*models.Template, error) {
	query := `
		SELECT id, template_id, name, severity, tags, author, file_path, created_at
		FROM templates
		WHERE id = ?
	`
	template := &models.Template{}
	err := d.db.QueryRow(query, id).Scan(
		&template.ID,
		&template.TemplateID,
		&template.Name,
		&template.Severity,
		&template.Tags,
		&template.Author,
		&template.FilePath,
		&template.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return template, nil
}

// GetTemplateByTemplateID retrieves a template by its template_id
func (d *Database) GetTemplateByTemplateID(templateID string) (*models.Template, error) {
	query := `
		SELECT id, template_id, name, severity, tags, author, file_path, created_at
		FROM templates
		WHERE template_id = ?
	`
	template := &models.Template{}
	err := d.db.QueryRow(query, templateID).Scan(
		&template.ID,
		&template.TemplateID,
		&template.Name,
		&template.Severity,
		&template.Tags,
		&template.Author,
		&template.FilePath,
		&template.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return template, nil
}

// GetAllTemplates retrieves all templates from the database
func (d *Database) GetAllTemplates() ([]*models.Template, error) {
	query := `
		SELECT id, template_id, name, severity, tags, author, file_path, created_at
		FROM templates
		ORDER BY created_at DESC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.Template
	for rows.Next() {
		template := &models.Template{}
		err := rows.Scan(
			&template.ID,
			&template.TemplateID,
			&template.Name,
			&template.Severity,
			&template.Tags,
			&template.Author,
			&template.FilePath,
			&template.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// SearchTemplates searches templates by name, tags, or severity
func (d *Database) SearchTemplates(keyword string, severity string) ([]*models.Template, error) {
	query := `
		SELECT id, template_id, name, severity, tags, author, file_path, created_at
		FROM templates
		WHERE (name LIKE ? OR tags LIKE ? OR template_id LIKE ?)
	`
	args := []interface{}{
		"%" + keyword + "%",
		"%" + keyword + "%",
		"%" + keyword + "%",
	}

	if severity != "" {
		query += " AND severity = ?"
		args = append(args, severity)
	}

	query += " ORDER BY severity DESC, created_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.Template
	for rows.Next() {
		template := &models.Template{}
		err := rows.Scan(
			&template.ID,
			&template.TemplateID,
			&template.Name,
			&template.Severity,
			&template.Tags,
			&template.Author,
			&template.FilePath,
			&template.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, template)
	}
	return templates, nil
}

// DeleteTemplate deletes a template by ID
func (d *Database) DeleteTemplate(id int64) error {
	query := "DELETE FROM templates WHERE id = ?"
	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

// BatchInsertTemplates inserts multiple templates at once
func (d *Database) BatchInsertTemplates(templates []*models.Template) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO templates (template_id, name, severity, tags, author, file_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, template := range templates {
		_, err := stmt.Exec(
			template.TemplateID,
			template.Name,
			template.Severity,
			template.Tags,
			template.Author,
			template.FilePath,
		)
		if err != nil {
			return fmt.Errorf("failed to insert template %s: %w", template.TemplateID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// ClearAllTemplates removes all templates from the database
func (d *Database) ClearAllTemplates() error {
	query := "DELETE FROM templates"
	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear templates: %w", err)
	}
	return nil
}
