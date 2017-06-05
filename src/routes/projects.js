if (process.env['NEWRELIC_LICENSE']) require('newrelic');
var express = require('express');
var passport = require('passport');
var request = require('request');
var multer  = require('multer');
var async = require('async');

var cinco = require("../lib/api");

var router = express.Router();

var storage = multer.diskStorage({
  destination: function (req, file, cb) {
    cb(null, 'public/uploads')
  },
  filename: function (req, file, cb) {
    cb(null, file.originalname)
  }
});
var upload = multer({ storage: storage });
var cpUpload = upload.fields([{ name: 'logo', maxCount: 1 }, { name: 'agreement', maxCount: 1 }]);

/*
  Projects:
  Resources to expose and manipulate details of projects
 */

 router.get('/project', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
   if(req.session.user.isAdmin || req.session.user.isProjectManager){
     var projManagerClient = cinco.client(req.session.user.cinco_keys);
     projManagerClient.getAllProjects(function (err, projects) {
       res.send(projects);
     });
   }
 });

 router.get('/project/status', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
   if(req.session.user.isAdmin || req.session.user.isProjectManager){
     var projManagerClient = cinco.client(req.session.user.cinco_keys);
     projManagerClient.getProjectStatuses(function (err, statuses) {
       res.send(statuses);
     });
   }
 });

 router.get('/project/categories', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
   if(req.session.user.isAdmin || req.session.user.isProjectManager){
     var projManagerClient = cinco.client(req.session.user.cinco_keys);
     projManagerClient.getProjectCategories(function (err, categories) {
       res.send(categories);
     });
   }
 });

 router.get('/project/sectors', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
   if(req.session.user.isAdmin || req.session.user.isProjectManager){
     var projManagerClient = cinco.client(req.session.user.cinco_keys);
     projManagerClient.getProjectSectors(function (err, sectors) {
       res.send(sectors);
     });
   }
 });

router.get('/projects', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getAllProjects(function (err, projects) {
      res.send(projects);
    });
  }
});

router.put('/projects/:projectId/managers', require('connect-ensure-login').ensureLoggedIn('/login'), cpUpload, function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projectId = req.params.projectId;
    var memberContact = req.body.managers;
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.updateProjectManagers(projectId, managers, function (err, updatedManagers) {
      return res.json(updatedManagers);
    });
  }
});

router.get('/projects/:id', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projectId = req.params.id;
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getProject(projectId, function (err, project) {
      // TODO: Create 404 page for when project doesn't exist
      if (err) return res.send('');
      res.send(project);
    });
  }
});

router.get('/projects/:projectId/config', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projectId = req.params.projectId;
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getProjectConfig(projectId, function (err, config) {
      // TODO: Create 404 page for when project doesn't exist
      if (err) return res.send('');
      res.send(config);
    });
  }
});

router.get('/create_project', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  res.render('create_project');
});

router.get('/my_projects', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getMyProjects(function (err, myProjects) {
      req.session.myProjects = myProjects;
      res.render('my_projects', {myProjects: myProjects});
    });
  }
});

router.get('/project/:id', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projectId = req.params.id;
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getProject(projectId, function (err, project) {
      // TODO: Create 404 page for when project doesn't exist
      if (err) return res.redirect('/');
      projManagerClient.getEmailAliases(projectId, function (err, emailAliases) {
        projManagerClient.getMailingListsAndParticipants(projectId, function (err, mailingLists) {
          projManagerClient.getMemberCompanies(projectId, function (err, memberCompanies) {
            async.forEach(memberCompanies, function (eachMember, callback){
              eachMember.orgName = "";
              eachMember.orgLogoRef = "";
              projManagerClient.getOrganization(eachMember.orgId, function (err, organization) {
                if(organization){
                  eachMember.orgName = organization.name;
                  eachMember.orgLogoRef = organization.logoRef;
                }
                callback();
              });
            }, function(err) {
              // Member Companies iteration done.
              return res.render('project', {project: project, emailAliases: emailAliases, mailingLists: mailingLists, memberCompanies:memberCompanies});
            });
          });
        });
      });
    });
  }
});

router.get('/archive_project/:id', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var id = req.params.id;
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.archiveProject(id, function (err) {
      return res.redirect('/');
    });
  }
});

router.post('/create_project', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    var now = new Date().toISOString();
    var url = req.body.url;
    if(url){
      if (!/^(?:f|ht)tps?\:\/\//.test(url)) url = "http://" + url;
    }
    var logoFileName = "";
    var agreementFileName = "";
    // if(req.files){
    //   if(req.files.logo) logoFileName = req.files.logo[0].originalname;
    //   if(req.files.agreement) agreementFileName = req.files.agreement[0].originalname;
    // }
    var newProject = {
      name: req.body.project_name,
      description: req.body.project_description,
      pm: req.session.user.user,
      url: url,
      startDate: now,
      logoRef: logoFileName,
      agreementRef: agreementFileName,
      category: req.body.project_type
    };
    projManagerClient.createProject(newProject, function (err, created, projectId) {
      var isNewAlias = req.body.isNewAlias;
      isNewAlias = (isNewAlias == "true");
      if(isNewAlias){
        var newAlias = JSON.parse(req.body.newAlias);
        async.forEach(newAlias, function (eachAlias, callback){
          projManagerClient.createEmailAliases(projectId, eachAlias, function (err, created, aliasId) {
            callback();
          });
        }, function(err) {
          // Email aliases iteration done.
          return res.redirect('/project/' + projectId);
        });
      }
      else{
        return res.redirect('/project/' + projectId);
      }
    });
  }
});

router.get('/edit_project/:projectId', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){

    var projectId = req.params.projectId;

    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getProject(projectId, function (err, project) {
      // TODO: Create 404 page for when project doesn't exist
      if (err) return res.redirect('/');
      project.domain = "";
      if(project.url) project.domain = project.url.replace('http://www.','').replace('https://www.','').replace('http://','').replace('/','');
      projManagerClient.getEmailAliases(projectId, function (err, emailAliases) {
        projManagerClient.getMemberCompanies(projectId, function (err, memberCompanies) {
          async.forEach(memberCompanies, function (eachMember, callback){
            eachMember.orgName = "";
            eachMember.orgLogoRef = "";
            projManagerClient.getOrganization(eachMember.orgId, function (err, organization) {
              if(organization){
                eachMember.orgName = organization.name;
                eachMember.orgLogoRef = organization.logoRef;
              }
              callback();
            });
          }, function(err) {
            // Member Companies iteration done.
            return res.render('edit_project', {project: project, emailAliases: emailAliases, memberCompanies:memberCompanies});
          });
        });
      });
    });

  }
});

router.post('/edit_project/:id', require('connect-ensure-login').ensureLoggedIn('/login'), cpUpload, function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){

    var id = req.params.id;

    // var logoFileName = "";
    // var agreementFileName = "";
    // var url = req.body.url;
    // if(url){
      // if (!/^(?:f|ht)tps?\:\/\//.test(url)) url = "http://" + url;
    // }
    // if(req.files.logo) logoFileName = req.files.logo[0].originalname;
    // else logoFileName = req.body.old_logoRef;
    // if(req.files.agreement) agreementFileName = req.files.agreement[0].originalname;
    // else agreementFileName = req.body.old_agreementRef;

    var updatedProps = {
      id: id,
      name: req.body.project_name,
      description: req.body.project_description,
      url: req.body.project_url,
      sector: req.body.project_sector,
      address: JSON.parse(req.body.project_address),
      status: req.body.project_status,
      category: req.body.project_category,
      startDate: req.body.project_start_date
      // pm: req.body.creator_pm,
      // logoRef: logoFileName,
      // agreementRef: agreementFileName,
    };
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.updateProject(updatedProps, function (err, updatedProject) {
      return res.json(updatedProject);
    });
  }
});


router.get('/get_all_projects', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getAllProjects(function (err, projects) {
      res.send(projects)
    });
  }
});

router.get('/get_project/:id', require('connect-ensure-login').ensureLoggedIn('/login'), function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projectId = req.params.id;
    if(req.query.members) { projectId = projectId + '?members=' + req.query.members }
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    projManagerClient.getProject(projectId, function (err, project) {
      // TODO: Create 404 page for when project doesn't exist
      if (err) return res.send('');
      res.send(project);
    });
  }
});

router.post('/post_project', require('connect-ensure-login').ensureLoggedIn('/login'), cpUpload, function(req, res){
  if(req.session.user.isAdmin || req.session.user.isProjectManager){
    var projManagerClient = cinco.client(req.session.user.cinco_keys);
    var now = new Date().toISOString();
    var url = req.body.url;
    if(url){
      if (!/^(?:f|ht)tps?\:\/\//.test(url)) url = "http://" + url;
    }
    var logoFileName = "";
    var agreementFileName = "";
    if(req.files){
      if(req.files.logo) logoFileName = req.files.logo[0].originalname;
      if(req.files.agreement) agreementFileName = req.files.agreement[0].originalname;
    }
    var newProject = {
      name: req.body.project_name,
      description: req.body.project_description,
      pm: req.session.user.user,
      url: url,
      startDate: now,
      logoRef: logoFileName,
      agreementRef: agreementFileName,
      category: req.body.project_type
    };
    projManagerClient.createProject(newProject, function (err, created, projectId) {
      return res.json(projectId);
    });
  }
});

module.exports = router;
