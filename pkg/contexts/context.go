package contexts

type IDbContext interface {
	IContext
}

type IContext interface {
	Begin() error
	Commit() error
	Rollback()
	Dispose()
}

type Context struct {
	DBContexts []IDbContext
}

func (c *Context) Base() *Context {
	return c
}

func (c *Context) AddDBContexts(dbContexts ...IDbContext) {
	c.DBContexts = append(c.DBContexts, dbContexts...)
}

func (c *Context) Begin() error {
	for _, dbContext := range c.DBContexts {
		if err := dbContext.Begin(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Commit() error {
	for _, dbContext := range c.DBContexts {
		if err := dbContext.Commit(); err != nil {
			return err
		}

	}
	return nil
}

func (c *Context) Rollback() {
	for _, dbContext := range c.DBContexts {
		dbContext.Rollback()
	}
}
func (c *Context) Dispose() {
	for _, dbContext := range c.DBContexts {
		dbContext.Dispose()
	}
}
