#### Packages ####
library(lme4)
library(DHARMa)
library(car)

#### Adding dataset ####
exp1 <- read.csv("exp1.csv", header = T)

exp1$reward<-factor(exp1$reward, levels = c("centre","A","D"))
exp1$group<-factor(exp1$group, levels = c("centre","border"))
exp1$phase<-factor(exp1$phase, levels = c("naive","trained"))

centre = subset(exp1, group == "centre")

border = subset(exp1, group == "border")

#### Analysis ####
##test for group interaction with training
feed.rewarding.interaction<-glmer(cbind(feed.rewarding,(feed.total-feed.rewarding)) ~ 
                                    phase * group + (1 | ID), 
                                  data=exp1, family = binomial)
simulationOutput <- simulateResiduals(fittedModel = feed.rewarding.interaction)
plot(simulationOutput)

summary(feed.rewarding.interaction)
Anova(feed.rewarding.interaction)

##group effect removed
feed.rewarding<-glmer(cbind(feed.rewarding,(feed.total-feed.rewarding)) ~ 
                        phase + (1 | ID), 
                      data=exp1, family = binomial)

simulationOutput <- simulateResiduals(fittedModel = feed.rewarding)
plot(simulationOutput)

summary(feed.rewarding)
Anova(feed.rewarding)

### Are butterflies trained to left row more likely to feed on feeders on the left half
### and vice versa?
feed.left.right <- glmer(cbind(feed.RW.side, feed.total-feed.RW.side) ~
                           phase + (1|ID),
                         data=border, family = binomial)

simulationOutput <- simulateResiduals(fittedModel = feed.left.right)
plot(simulationOutput)

summary(feed.left.right)
Anova(feed.left.right)

boxplot(data = border, (feed.RW.side/feed.total)~phase)

### are border butterflies learning left or right or border or middle?
trained.border <- subset(border, phase == "trained")

feed.trained.border <- glm(cbind(feed.A, feed.D) ~
                          reward,
                        data=trained.border, family = binomial)

simulationOutput <- simulateResiduals(fittedModel = feed.trained.border)
plot(simulationOutput)

summary(feed.trained.border)
Anova(feed.trained.border)

boxplot(data = trained.border, (feed.D/feed.A)~reward)

### are border butterflies more likely to feed on border flowers, even if not rewarding, test if ratio of
### centre to non-rewawrding border flowers changes with training
feed.nonreward.border <- glmer(cbind(feed.nonreward.edge, feed.centre) ~
                                                      phase + (1|ID) ,
                                                    data=border, family = binomial)

simulationOutput <- simulateResiduals(fittedModel = feed.nonreward.border)
plot(simulationOutput)

summary(feed.nonreward.border)
Anova(feed.nonreward.border)

boxplot(data = border, (feed.nonreward.edge/(feed.nonreward.edge+feed.centre))~phase)

## are border butterflies learning A or D, over B1,C1,B4,C4?
feed.A.D <- glmer(cbind(feed.A.D, feed.B1.C1.B4.C4) ~
                                 phase + (1|ID),
                               data=border, family = binomial)

boxplot(data = exp1, (feed.B1.C1.B4.C4/(feed.A.D+feed.B1.C1.B4.C4))~phase*group)

## Does trained performance differ between days?
Exp1byDay <- read.csv("Exp1byDay.csv")

TrainedbyDay <- subset(Exp1byDay, phase != "naive")

byDay.model <- glmer(cbind(feed.rewarding, (total-feed.rewarding)) ~
                       phase + (1|ID) ,
                     data=TrainedbyDay, family = binomial)

simulationOutput <- simulateResiduals(fittedModel = byDay.model)
plot(simulationOutput)

summary(byDay.model)
Anova(byDay.model)

boxplot(data = TrainedbyDay, feed.rewarding/total~phase)
Exp1byDay$feed.rewarding
