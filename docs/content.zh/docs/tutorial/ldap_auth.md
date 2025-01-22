---
title: "LDAP 身份认证集成"
weight: 10
draft: true
---

# LDAP 身份认证集成

## LDAP 简介

LDAP（轻量级目录访问协议,Lightweight Directory Access Protocol)是实现提供被称为目录服务的信息服务.目录服务是一种特殊的数据库系统,其专门针对读取,浏览和搜索操作进行了特定的优化.
目录一般用来包含描述性的,基于属性的信息并支持精细复杂的过滤能力.目录一般不支持通用数据库针对大量更新操作操作需要的复杂的事务管理或回卷策略.
而目录服务的更新则一般都非常简单.这种目录可以存储包括个人信息,web 链结,jpeg 图像等各种信息.为了访问存储在目录中的信息,就需要使用运行在 TCP/IP 之上的访问协议—LDAP.

LDAP 目录中的信息是是按照树型结构组织,具体信息存储在条目(entry)的数据结构中.条目相当于关系数据库中表的记录;条目是具有区别名 DN （Distinguished Name）的属性（Attribute）,DN 是用来引用条目的,DN 相当于关系数据库表中的关键字（Primary Key）.属性由类型（Type）和一个或多个值（Values）组成,相当于关系数据库中的字段（Field）由字段名和数据类型组成,只是为了方便检索的需要,LDAP 中的 Type 可以有多个 Value,而不是关系数据库中为降低数据的冗余性要求实现的各个域必须是不相关的.LDAP 中条目的组织一般按照地理位置和组织关系进行组织,非常的直观.LDAP 把数据存放在文件中,为提高效率可以使用基于索引的文件数据库,而不是关系数据库.类型的一个例子就是 mail,其值将是一个电子邮件地址.

LDAP 的信息是以树型结构存储的,在树根一般定义国家(c=CN)或域名(dc=com),在其下则往往定义一个或多个组织 (organization)(o=Acme)或组织单元(organizational units) (ou=People).一个组织单元可能包含诸如所有雇员,大楼内的所有打印机等信息.此外,LDAP 支持对条目能够和必须支持哪些属性进行控制,这是有一个特殊的称为对象类别(objectClass)的属性来实现的.该属性的值决定了该条目必须遵循的一些规则,其规定了该条目能够及至少应该包含哪些属性.例如：inetorgPerson 对象类需要支持 sn(surname)和 cn(common name)属性,但也可以包含可选的如邮件,电话号码等属性.

## LDAP 简称对应

- o– organization（组织-公司）
- ou – organization unit（组织单元-部门）
- c - countryName（国家）
- dc - domainComponent（域名）
- sn – suer name（真实名称）
- cn - common name（常用名称）
