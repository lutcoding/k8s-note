// 定义一个权限管理接口
type PermissionManager interface {
    // 获取用户的角色
    GetRoles(user string) ([]string, error)
    // 设置用户的角色
    SetRoles(user string, roles []string) error
    // 获取用户的权限
    GetPermissions(user string) ([]string, error)
    // 设置用户的权限
    SetPermissions(user string, permissions []string) error
    // 检查用户是否有某个权限
    CheckPermission(user string, permission string) (bool, error)
}

// 实现一个基于casbin的权限管理接口
type CasbinPermissionManager struct {
    // casbin的enforcer对象
    enforcer *casbin.Enforcer
}

// 创建一个CasbinPermissionManager对象
func NewCasbinPermissionManager(enforcer *casbin.Enforcer) *CasbinPermissionManager {
    return &CasbinPermissionManager{enforcer: enforcer}
}

// 获取用户的角色
func (c *CasbinPermissionManager) GetRoles(user string) ([]string, error) {
    return c.enforcer.GetRolesForUser(user)
}

// 设置用户的角色
func (c *CasbinPermissionManager) SetRoles(user string, roles []string) error {
    // 删除用户的所有角色
    _, err := c.enforcer.DeleteRolesForUser(user)
    if err != nil {
        return err
    }
    // 添加用户的新角色
    for _, role := range roles {
        _, err = c.enforcer.AddRoleForUser(user, role)
        if err != nil {
            return err
        }
    }
    return nil
}

// 获取用户的权限
func (c *CasbinPermissionManager) GetPermissions(user string) ([]string, error) {
    // 获取用户的所有策略
    policies := c.enforcer.GetFilteredPolicy(0, user)
    // 将策略转换为权限字符串
    permissions := make([]string, len(policies))
    for i, policy := range policies {
        permissions[i] = strings.Join(policy[1:], ":")
    }
    return permissions, nil
}

// 设置用户的权限
func (c *CasbinPermissionManager) SetPermissions(user string, permissions []string) error {
    // 删除用户的所有策略
    _, err := c.enforcer.DeletePermissionsForUser(user)
    if err != nil {
        return err
    }
    // 添加用户的新策略
    for _, permission := range permissions {
        // 将权限字符串分割为策略元素
        elements := strings.Split(permission, ":")
        // 在策略元素前加上用户
        elements = append([]string{user}, elements...)
        // 添加策略
        _, err = c.enforcer.AddPolicy(elements...)
        if err != nil {
            return err
        }
    }
    return nil
}

// 检查用户是否有某个权限
func (c *CasbinPermissionManager) CheckPermission(user string, permission string) (bool, error) {
    // 将权限字符串分割为策略元素
    elements := strings.Split(permission, ":")
    // 检查用户是否有该策略
    return c.enforcer.Enforce(user, elements...)
}
