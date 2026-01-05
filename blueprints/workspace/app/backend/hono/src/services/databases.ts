import type { Store } from '../store/types';
import type { Database, Property, CreateDatabase, UpdateDatabase, AddProperty, UpdateProperty } from '../models';
import { generateId } from '../utils/id';
import { defaultDatabaseProperties } from '../models/database';

export class DatabaseService {
  constructor(private store: Store) {}

  async create(input: CreateDatabase, userId: string): Promise<{ database: Database; page: import('../models').Page }> {
    // Create the page that will host this database
    const page = await this.store.pages.create({
      id: generateId(),
      workspaceId: input.workspaceId,
      parentId: input.pageId,
      parentType: input.pageId ? 'page' : 'workspace',
      title: input.title ?? 'Untitled Database',
      icon: input.icon,
      isTemplate: false,
      isArchived: false,
      createdBy: userId,
      properties: {},
    });

    // Create the database
    const database = await this.store.databases.create({
      id: generateId(),
      workspaceId: input.workspaceId,
      pageId: page.id,
      title: input.title ?? 'Untitled Database',
      icon: input.icon,
      isInline: input.isInline ?? false,
      properties: input.properties ?? defaultDatabaseProperties(),
    });

    return { database, page };
  }

  async getById(id: string): Promise<Database | null> {
    return this.store.databases.getById(id);
  }

  async getByPageId(pageId: string): Promise<Database | null> {
    return this.store.databases.getByPageId(pageId);
  }

  async listByWorkspace(workspaceId: string): Promise<Database[]> {
    return this.store.databases.listByWorkspace(workspaceId);
  }

  async update(id: string, data: UpdateDatabase): Promise<Database> {
    return this.store.databases.update(id, data);
  }

  async delete(id: string): Promise<void> {
    const database = await this.store.databases.getById(id);
    if (!database) {
      throw new Error('Database not found');
    }

    // Delete all rows (pages with this database_id)
    // Note: This would be handled by cascade in a real implementation

    // Delete views
    const views = await this.store.views.listByDatabase(id);
    for (const view of views) {
      await this.store.views.delete(view.id);
    }

    // Delete the database
    await this.store.databases.delete(id);

    // Delete the host page
    await this.store.pages.delete(database.pageId);
  }

  async addProperty(id: string, input: AddProperty): Promise<Database> {
    const database = await this.store.databases.getById(id);
    if (!database) {
      throw new Error('Database not found');
    }

    const property: Property = {
      id: generateId(),
      name: input.name,
      type: input.type,
      config: input.config,
    };

    const properties = [...database.properties, property];
    return this.store.databases.update(id, { properties });
  }

  async updateProperty(id: string, propertyId: string, input: UpdateProperty): Promise<Database> {
    const database = await this.store.databases.getById(id);
    if (!database) {
      throw new Error('Database not found');
    }

    const properties = database.properties.map((prop) => {
      if (prop.id === propertyId) {
        return {
          ...prop,
          name: input.name ?? prop.name,
          config: input.config ?? prop.config,
        };
      }
      return prop;
    });

    return this.store.databases.update(id, { properties });
  }

  async deleteProperty(id: string, propertyId: string): Promise<Database> {
    const database = await this.store.databases.getById(id);
    if (!database) {
      throw new Error('Database not found');
    }

    // Cannot delete title property
    const property = database.properties.find((p) => p.id === propertyId);
    if (property?.type === 'title') {
      throw new Error('Cannot delete title property');
    }

    const properties = database.properties.filter((prop) => prop.id !== propertyId);
    return this.store.databases.update(id, { properties });
  }
}
